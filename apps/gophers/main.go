// SPDX-License-Identifier: Unlicense OR MIT

package main

// A Gio program that displays Go contributors from GitHub. See https://gioui.org for more information.

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"net/http"
	"os"
	"runtime"

	"golang.org/x/image/draw"
	"golang.org/x/oauth2"

	_ "image/jpeg"
	_ "image/png"

	_ "net/http/pprof"

	"gioui.org/ui"
	"gioui.org/ui/app"
	gdraw "gioui.org/ui/draw"
	"gioui.org/ui/f32"
	"gioui.org/ui/gesture"
	"gioui.org/ui/input"
	"gioui.org/ui/key"
	"gioui.org/ui/layout"
	"gioui.org/ui/measure"
	"gioui.org/ui/pointer"
	"gioui.org/ui/system"
	"gioui.org/ui/text"
	"gioui.org/ui/widget"
	"golang.org/x/exp/shiny/iconvg"

	"github.com/google/go-github/v24/github"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"golang.org/x/exp/shiny/materialdesign/icons"
)

type App struct {
	w     *app.Window
	faces measure.Faces

	fab *ActionButton

	usersList   *layout.List
	edit, edit2 *text.Editor

	users        []*user
	userClicks   []gesture.Click
	selectedUser *userPage

	updateUsers chan []*user

	ctx       context.Context
	ctxCancel context.CancelFunc

	// Profiling.
	profiling   bool
	profile     system.ProfileEvent
	lastMallocs uint64
}

type userPage struct {
	config        ui.Config
	faces         measure.Faces
	invalidate    func()
	user          *user
	commitsList   *layout.List
	commits       []*github.Commit
	commitsResult chan []*github.Commit
}

type user struct {
	name    string
	login   string
	company string
	avatar  image.Image
}

type icon struct {
	src  []byte
	size ui.Value

	// Cached values.
	img     image.Image
	imgSize int
}

type ActionButton struct {
	face    text.Face
	Open    bool
	icons   []*icon
	sendIco *icon
}

var (
	profile = flag.Bool("profile", false, "serve profiling data at http://localhost:6060")
	stats   = flag.Bool("stats", false, "show rendering statistics")
	token   = flag.String("token", "", "Github authentication token")
)

var fonts struct {
	regular *sfnt.Font
	bold    *sfnt.Font
	italic  *sfnt.Font
	mono    *sfnt.Font
}

var theme struct {
	text     ui.MacroOp
	tertText ui.MacroOp
	brand    ui.MacroOp
	white    ui.MacroOp
}

func main() {
	flag.Parse()
	initProfiling()
	if *token == "" {
		fmt.Println("The quota for anonymous GitHub API access is very low. Specify a token with -token to avoid quota errors.")
		fmt.Println("See https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line.")
	}
	go func() {
		fonts.regular = mustLoadFont(goregular.TTF)
		fonts.bold = mustLoadFont(gobold.TTF)
		fonts.italic = mustLoadFont(goitalic.TTF)
		fonts.mono = mustLoadFont(gomono.TTF)
		var ops ui.Ops
		theme.text = colorMaterial(&ops, rgb(0x333333))
		theme.tertText = colorMaterial(&ops, rgb(0xbbbbbb))
		theme.brand = colorMaterial(&ops, rgb(0x62798c))
		theme.white = colorMaterial(&ops, rgb(0xffffff))
		w := app.NewWindow(&app.WindowOptions{
			Width:  ui.Dp(400),
			Height: ui.Dp(800),
			Title:  "Gophers",
		})
		if err := newApp(w).run(); err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()
}

func initProfiling() {
	if !*profile {
		return
	}
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

func colorMaterial(ops *ui.Ops, color color.RGBA) ui.MacroOp {
	var mat ui.MacroOp
	mat.Record(ops)
	gdraw.ColorOp{Color: color}.Add(ops)
	mat.Stop()
	return mat
}

func (a *App) run() error {
	a.profiling = *stats
	ops := new(ui.Ops)
	var cfg app.Config
	for {
		select {
		case users := <-a.updateUsers:
			a.users = users
			a.userClicks = make([]gesture.Click, len(users))
			a.w.Invalidate()
		case e := <-a.w.Events():
			switch e := e.(type) {
			case key.ChordEvent:
				switch e.Name {
				case key.NameEscape:
					os.Exit(0)
				case 'P':
					if e.Modifiers.Contain(key.ModCommand) {
						a.profiling = !a.profiling
						a.w.Invalidate()
					}
				}
			case app.DestroyEvent:
				return e.Err
			case app.StageEvent:
				if e.Stage >= app.StageRunning {
					if a.ctxCancel == nil {
						a.ctx, a.ctxCancel = context.WithCancel(context.Background())
					}
					if a.users == nil {
						go a.fetchContributors()
					}
				} else {
					if a.ctxCancel != nil {
						a.ctxCancel()
						a.ctxCancel = nil
					}
				}
			case *app.CommandEvent:
				switch e.Type {
				case app.CommandBack:
					if a.selectedUser != nil {
						a.selectedUser = nil
						e.Cancel = true
						a.w.Invalidate()
					}
				}
			case app.DrawEvent:
				ops.Reset()
				a.faces.Reset(&cfg)
				cfg = e.Config
				cs := layout.RigidConstraints(e.Size)
				a.Layout(&cfg, a.w.Queue(), ops, cs)
				if a.profiling {
					a.layoutTimings(&cfg, a.w.Queue(), ops, cs)
				}
				a.w.Draw(ops)
			}
		}
	}
}

func newApp(w *app.Window) *App {
	a := &App{
		w:           w,
		updateUsers: make(chan []*user),
	}
	a.usersList = &layout.List{
		Axis: layout.Vertical,
	}
	a.fab = &ActionButton{
		face:    a.face(fonts.regular, 11),
		sendIco: &icon{src: icons.ContentSend, size: ui.Dp(24)},
		icons:   []*icon{},
	}
	a.edit2 = &text.Editor{
		Face: a.face(fonts.italic, 14),
		//Alignment: text.End,
		SingleLine:   true,
		Hint:         "Hint",
		HintMaterial: theme.tertText,
		Material:     theme.text,
	}
	a.edit2.SetText("Single line editor. Edit me!")
	a.edit = &text.Editor{
		Face:     a.face(fonts.regular, 16),
		Material: theme.text,
		//Alignment: text.End,
		//SingleLine: true,
	}
	a.edit.SetText(longTextSample)
	return a
}

func githubClient(ctx context.Context) *github.Client {
	var tc *http.Client
	if *token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}
	return github.NewClient(tc)
}

func (a *App) fetchContributors() {
	client := githubClient(a.ctx)
	cons, _, err := client.Repositories.ListContributors(a.ctx, "golang", "go", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "github: failed to fetch contributors: %v\n", err)
		return
	}
	var users []*user
	userErrs := make(chan error, len(cons))
	avatarErrs := make(chan error, len(cons))
	for _, con := range cons {
		con := con
		avatar := con.GetAvatarURL()
		if avatar == "" {
			continue
		}
		u := &user{
			login: con.GetLogin(),
		}
		users = append(users, u)
		go func() {
			guser, _, err := client.Users.Get(a.ctx, u.login)
			if err != nil {
				avatarErrs <- err
				return
			}
			u.name = guser.GetName()
			u.company = guser.GetCompany()
			avatarErrs <- nil
		}()
		go func() {
			a, err := fetchImage(avatar)
			u.avatar = a
			userErrs <- err
		}()
	}
	for i := 0; i < len(cons); i++ {
		if err := <-userErrs; err != nil {
			fmt.Fprintf(os.Stderr, "github: failed to fetch user: %v\n", err)
		}
		if err := <-avatarErrs; err != nil {
			fmt.Fprintf(os.Stderr, "github: failed to fetch avatar: %v\n", err)
		}
	}
	// Drop users with no avatar or name.
	for i := len(users) - 1; i >= 0; i-- {
		if u := users[i]; u.name == "" || u.avatar == nil {
			users = append(users[:i], users[i+1:]...)
		}
	}
	a.updateUsers <- users
}

func fetchImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetchImage: http.Get(%q): %v", url, err)
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchImage: image decode failed: %v", err)
	}
	return img, nil
}

func mustLoadFont(fontData []byte) *sfnt.Font {
	fnt, err := sfnt.Parse(fontData)
	if err != nil {
		panic("failed to load font")
	}
	return fnt
}

func rgb(c uint32) color.RGBA {
	return argb((0xff << 24) | c)
}

func argb(c uint32) color.RGBA {
	return color.RGBA{A: uint8(c >> 24), R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c)}
}

func (a *App) face(f *sfnt.Font, size float32) text.Face {
	return a.faces.For(f, ui.Sp(size))
}

func (a *App) layoutTimings(c ui.Config, q input.Queue, ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	for _, e := range q.Events(a) {
		if e, ok := e.(system.ProfileEvent); ok {
			a.profile = e
		}
	}
	system.ProfileOp{Key: a}.Add(ops)
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	mallocs := mstats.Mallocs - a.lastMallocs
	a.lastMallocs = mstats.Mallocs
	al := layout.Align{Alignment: layout.NE}
	cs = al.Begin(ops, cs)
	in := layout.Inset{Top: ui.Dp(16)}
	cs = in.Begin(c, ops, cs)
	txt := fmt.Sprintf("m: %d %s", mallocs, a.profile.Timings)
	dims := text.Label{Material: theme.text, Face: a.face(fonts.mono, 10), Text: txt}.Layout(ops, cs)
	dims = in.End(dims)
	return al.End(dims)
}

func (a *App) Layout(c ui.Config, q input.Queue, ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	for i := range a.userClicks {
		click := &a.userClicks[i]
		for _, e := range click.Events(q) {
			if e.Type == gesture.TypeClick {
				a.selectedUser = a.newUserPage(a.users[i])
			}
		}
	}
	if a.selectedUser == nil {
		return a.layoutUsers(c, q, ops, cs)
	} else {
		return a.selectedUser.Layout(c, q, ops, cs)
	}
}

func (a *App) newUserPage(user *user) *userPage {
	up := &userPage{
		faces:         a.faces,
		invalidate:    a.w.Invalidate,
		user:          user,
		commitsList:   &layout.List{Axis: layout.Vertical},
		commitsResult: make(chan []*github.Commit, 1),
	}
	up.fetchCommits(a.ctx)
	return up
}

func (up *userPage) Layout(c ui.Config, q input.Queue, ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	l := up.commitsList
	if l.Dragging() {
		key.HideInputOp{}.Add(ops)
	}
	select {
	case commits := <-up.commitsResult:
		up.commits = commits
	default:
	}
	for l.Init(c, q, ops, cs, len(up.commits)); l.More(); l.Next() {
		l.Elem(up.commit(c, ops, l.Constraints(), l.Index()))
	}
	return l.Layout()
}

func (up *userPage) commit(c ui.Config, ops *ui.Ops, cs layout.Constraints, index int) layout.Dimens {
	u := up.user
	msg := up.commits[index].GetMessage()
	label := text.Label{Material: theme.text, Face: up.faces.For(fonts.regular, ui.Sp(12)), Text: msg}
	in := layout.Inset{Top: ui.Dp(16), Right: ui.Dp(8), Left: ui.Dp(8)}
	cs = in.Begin(c, ops, cs)
	f := (&layout.Flex{Axis: layout.Horizontal, MainAxisAlignment: layout.Start, CrossAxisAlignment: layout.Start}).Init(ops, cs)
	cs = f.Rigid()
	sz := c.Px(ui.Dp(48))
	cc := clipCircle{}
	cs = cc.Begin(ops, cs)
	cs = layout.RigidConstraints(cs.Constrain(image.Point{X: sz, Y: sz}))
	dims := widget.Image{Src: u.avatar, Rect: u.avatar.Bounds()}.Layout(c, ops, cs)
	dims = cc.End(dims)
	c1 := f.End(dims)
	cs = f.Flexible(1)
	cs.Width.Min = cs.Width.Max
	in2 := layout.Inset{Left: ui.Dp(8)}
	cs = in2.Begin(c, ops, cs)
	dims = label.Layout(ops, cs)
	dims = in2.End(dims)
	c2 := f.End(dims)
	dims = f.Layout(c1, c2)
	return in.End(dims)
}

func (up *userPage) fetchCommits(ctx context.Context) {
	go func() {
		gh := githubClient(ctx)
		repoCommits, _, err := gh.Repositories.ListCommits(ctx, "golang", "go", &github.CommitsListOptions{
			Author: up.user.login,
		})
		if err != nil {
			log.Printf("failed to fetch commits: %v", err)
			return
		}
		var commits []*github.Commit
		for _, commit := range repoCommits {
			if c := commit.GetCommit(); c != nil {
				commits = append(commits, c)
			}
		}
		up.commitsResult <- commits
		up.invalidate()
	}()
}

func (a *App) layoutUsers(c ui.Config, q input.Queue, ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	st := (&layout.Stack{Alignment: layout.Start}).Init(ops, cs)
	cs = st.Rigid()
	al := layout.Align{Alignment: layout.SE}
	in := layout.UniformInset(ui.Dp(16))
	cs = in.Begin(c, ops, al.Begin(ops, cs))
	dims := a.fab.Layout(c, q, ops, cs)
	dims = al.End(in.End(dims))
	c2 := st.End(dims)

	cs = st.Expand()
	{
		f := (&layout.Flex{Axis: layout.Vertical, MainAxisAlignment: layout.Start}).Init(ops, cs)

		cs = f.Rigid()
		{
			cs.Width.Min = cs.Width.Max
			in := layout.UniformInset(ui.Dp(16))
			sz := c.Px(ui.Dp(200))
			cs = layout.RigidConstraints(cs.Constrain(image.Point{X: sz, Y: sz}))
			dims = a.edit.Layout(c, q, ops, in.Begin(c, ops, cs))
			dims = in.End(dims)
		}
		c1 := f.End(dims)

		cs = f.Rigid()
		{
			cs.Width.Min = cs.Width.Max
			in := layout.Inset{Bottom: ui.Dp(16), Left: ui.Dp(16), Right: ui.Dp(16)}
			dims = a.edit2.Layout(c, q, ops, in.Begin(c, ops, cs))
			dims = in.End(dims)
		}
		c2 := f.End(dims)

		cs = f.Rigid()
		{
			cs.Width.Min = cs.Width.Max
			s := layout.Stack{Alignment: layout.Center}
			s.Init(ops, cs)
			cs = s.Rigid()
			in := layout.Inset{Top: ui.Dp(16), Right: ui.Dp(8), Bottom: ui.Dp(8), Left: ui.Dp(8)}
			grey := colorMaterial(ops, rgb(0x888888))
			lbl := text.Label{Material: grey, Face: a.face(fonts.regular, 11), Text: "GOPHERS"}
			dims = in.End(lbl.Layout(ops, in.Begin(c, ops, cs)))
			c2 := s.End(dims)
			c1 := s.End(fill{colorMaterial(ops, rgb(0xf2f2f2))}.Layout(ops, s.Expand()))
			dims = s.Layout(c1, c2)
		}
		c3 := f.End(dims)

		cs = f.Flexible(1)
		cs.Width.Min = cs.Width.Max
		dims = a.layoutContributors(c, q, ops, cs)
		c4 := f.End(dims)
		dims = f.Layout(c1, c2, c3, c4)
	}
	c1 := st.End(dims)
	return st.Layout(c1, c2)
}

func (a *ActionButton) Layout(c ui.Config, q input.Queue, ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	f := layout.Flex{Axis: layout.Vertical, MainAxisAlignment: layout.Start, CrossAxisAlignment: layout.End}
	f.Init(ops, cs)
	cs = f.Rigid()
	in := layout.Inset{Top: ui.Dp(4)}
	cs = in.Begin(c, ops, cs)
	dims := fab(ops, cs, a.sendIco.image(c), theme.brand, c.Px(ui.Dp(56)))
	pointer.EllipseAreaOp{Size: dims.Size}.Add(ops)
	dims = in.End(dims)
	return f.Layout(f.End(dims))
}

func (a *App) layoutContributors(c ui.Config, q input.Queue, ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	l := a.usersList
	if l.Dragging() {
		key.HideInputOp{}.Add(ops)
	}
	for l.Init(c, q, ops, cs, len(a.users)); l.More(); l.Next() {
		l.Elem(a.user(c, ops, l.Constraints(), l.Index()))
	}
	return l.Layout()
}

func (a *App) user(c ui.Config, ops *ui.Ops, cs layout.Constraints, index int) layout.Dimens {
	u := a.users[index]
	elem := layout.Flex{Axis: layout.Vertical, MainAxisAlignment: layout.Start, CrossAxisAlignment: layout.Start}
	elem.Init(ops, cs)
	cs = elem.Rigid()
	var dims layout.Dimens
	{
		in := layout.UniformInset(ui.Dp(8))
		cs = in.Begin(c, ops, cs)
		f := centerRowOpts()
		f.Init(ops, cs)
		cs = f.Rigid()
		{
			in := layout.Inset{Right: ui.Dp(8)}
			cc := clipCircle{}
			cs = cc.Begin(ops, in.Begin(c, ops, cs))
			sz := image.Point{X: c.Px(ui.Dp(48)), Y: c.Px(ui.Dp(48))}
			cs = layout.RigidConstraints(cs.Constrain(sz))
			dims = widget.Image{Src: u.avatar, Rect: u.avatar.Bounds()}.Layout(c, ops, cs)
			dims = in.End(cc.End(dims))
		}
		c1 := f.End(dims)
		cs = f.Rigid()
		{
			f := column()
			f.Init(ops, cs)
			cs = f.Rigid()
			{
				f := baseline()
				f.Init(ops, cs)
				cs = f.Rigid()
				dims = text.Label{Material: theme.text, Face: a.face(fonts.regular, 13), Text: u.name}.Layout(ops, cs)
				c1 := f.End(dims)
				cs = f.Flexible(1)
				cs.Width.Min = cs.Width.Max
				al := layout.Align{Alignment: layout.E}
				in := layout.Inset{Left: ui.Dp(2)}
				cs = in.Begin(c, ops, al.Begin(ops, cs))
				dims = text.Label{Material: theme.text, Face: a.face(fonts.regular, 10), Text: "3 hours ago"}.Layout(ops, cs)
				dims = al.End(in.End(dims))
				c2 := f.End(dims)
				dims = f.Layout(c1, c2)
			}
			c1 := f.End(dims)
			cs = f.Rigid()
			in := layout.Inset{Top: ui.Dp(4)}
			cs = in.Begin(c, ops, cs)
			dims = text.Label{Material: theme.tertText, Face: a.face(fonts.regular, 12), Text: u.company}.Layout(ops, cs)
			dims = in.End(dims)
			c2 := f.End(dims)
			dims = f.Layout(c1, c2)
		}
		c2 := f.End(dims)
		dims = f.Layout(c1, c2)
		dims = in.End(dims)
		pointer.RectAreaOp{Size: dims.Size}.Add(ops)
		click := &a.userClicks[index]
		click.Add(ops)
	}
	c1 := elem.End(dims)
	return elem.Layout(c1)
}

type fill struct {
	material ui.MacroOp
}

func (f fill) Layout(ops *ui.Ops, cs layout.Constraints) layout.Dimens {
	d := image.Point{X: cs.Width.Max, Y: cs.Height.Max}
	if d.X == ui.Inf {
		d.X = cs.Width.Min
	}
	if d.Y == ui.Inf {
		d.Y = cs.Height.Min
	}
	dr := f32.Rectangle{
		Max: f32.Point{X: float32(d.X), Y: float32(d.Y)},
	}
	f.material.Add(ops)
	gdraw.DrawOp{Rect: dr}.Add(ops)
	return layout.Dimens{Size: d, Baseline: d.Y}
}

func column() layout.Flex {
	return layout.Flex{Axis: layout.Vertical, MainAxisAlignment: layout.Start, CrossAxisAlignment: layout.Start}
}

func centerRowOpts() layout.Flex {
	return layout.Flex{Axis: layout.Horizontal, MainAxisAlignment: layout.Start, CrossAxisAlignment: layout.Center}
}

func baseline() layout.Flex {
	return layout.Flex{Axis: layout.Horizontal, CrossAxisAlignment: layout.Baseline}
}

type clipCircle struct {
	m   ui.MacroOp
	ops *ui.Ops
}

func (c *clipCircle) Begin(ops *ui.Ops, cs layout.Constraints) layout.Constraints {
	c.ops = ops
	c.m.Record(ops)
	return cs
}

func (c *clipCircle) End(dims layout.Dimens) layout.Dimens {
	c.m.Stop()
	ops := c.ops
	max := dims.Size.X
	if dy := dims.Size.Y; dy > max {
		max = dy
	}
	szf := float32(max)
	rr := szf * .5
	var stack ui.StackOp
	stack.Push(ops)
	rrect(ops, szf, szf, rr, rr, rr, rr)
	c.m.Add(ops)
	stack.Pop()
	return dims
}

func fab(ops *ui.Ops, cs layout.Constraints, ico image.Image, mat ui.MacroOp, size int) layout.Dimens {
	dp := image.Point{X: (size - ico.Bounds().Dx()) / 2, Y: (size - ico.Bounds().Dy()) / 2}
	dims := image.Point{X: size, Y: size}
	rr := float32(size) * .5
	rrect(ops, float32(size), float32(size), rr, rr, rr, rr)
	mat.Add(ops)
	gdraw.DrawOp{Rect: f32.Rectangle{Max: f32.Point{X: float32(size), Y: float32(size)}}}.Add(ops)
	gdraw.ImageOp{Src: ico, Rect: ico.Bounds()}.Add(ops)
	gdraw.DrawOp{
		Rect: toRectF(ico.Bounds().Add(dp)),
	}.Add(ops)
	return layout.Dimens{Size: dims}
}

func toRectF(r image.Rectangle) f32.Rectangle {
	return f32.Rectangle{
		Min: f32.Point{X: float32(r.Min.X), Y: float32(r.Min.Y)},
		Max: f32.Point{X: float32(r.Max.X), Y: float32(r.Max.Y)},
	}
}

func (ic *icon) image(cfg ui.Config) image.Image {
	sz := cfg.Px(ic.size)
	if sz == ic.imgSize {
		return ic.img
	}
	m, _ := iconvg.DecodeMetadata(ic.src)
	dx, dy := m.ViewBox.AspectRatio()
	img := image.NewRGBA(image.Rectangle{Max: image.Point{X: sz, Y: int(float32(sz) * dy / dx)}})
	var ico iconvg.Rasterizer
	ico.SetDstImage(img, img.Bounds(), draw.Src)
	// Use white for icons.
	m.Palette[0] = color.RGBA{A: 0xff, R: 0xff, G: 0xff, B: 0xff}
	iconvg.Decode(&ico, ic.src, &iconvg.DecodeOptions{
		Palette: &m.Palette,
	})
	ic.img = img
	ic.imgSize = sz
	return img
}

// https://pomax.github.io/bezierinfo/#circles_cubic.
func rrect(ops *ui.Ops, width, height, se, sw, nw, ne float32) {
	w, h := float32(width), float32(height)
	const c = 0.55228475 // 4*(sqrt(2)-1)/3
	var b gdraw.PathBuilder
	b.Init(ops)
	b.Move(f32.Point{X: w, Y: h - se})
	b.Cube(f32.Point{X: 0, Y: se * c}, f32.Point{X: -se + se*c, Y: se}, f32.Point{X: -se, Y: se}) // SE
	b.Line(f32.Point{X: sw - w + se, Y: 0})
	b.Cube(f32.Point{X: -sw * c, Y: 0}, f32.Point{X: -sw, Y: -sw + sw*c}, f32.Point{X: -sw, Y: -sw}) // SW
	b.Line(f32.Point{X: 0, Y: nw - h + sw})
	b.Cube(f32.Point{X: 0, Y: -nw * c}, f32.Point{X: nw - nw*c, Y: -nw}, f32.Point{X: nw, Y: -nw}) // NW
	b.Line(f32.Point{X: w - ne - nw, Y: 0})
	b.Cube(f32.Point{X: ne * c, Y: 0}, f32.Point{X: ne, Y: ne - ne*c}, f32.Point{X: ne, Y: ne}) // NE
	b.End()
}

const longTextSample = `1. I learned from my grandfather, Verus, to use good manners, and to
put restraint on anger. 2. In the famous memory of my father I had a
pattern of modesty and manliness. 3. Of my mother I learned to be
pious and generous; to keep myself not only from evil deeds, but even
from evil thoughts; and to live with a simplicity which is far from
customary among the rich. 4. I owe it to my great-grandfather that I
did not attend public lectures and discussions, but had good and able
teachers at home; and I owe him also the knowledge that for things of
this nature a man should count no expense too great.

5. My tutor taught me not to favour either green or blue at the
chariot races, nor, in the contests of gladiators, to be a supporter
either of light or heavy armed. He taught me also to endure labour;
not to need many things; to serve myself without troubling others; not
to intermeddle in the affairs of others, and not easily to listen to
slanders against them.

6. Of Diognetus I had the lesson not to busy myself about vain things;
not to credit the great professions of such as pretend to work
wonders, or of sorcerers about their charms, and their expelling of
Demons and the like; not to keep quails (for fighting or divination),
nor to run after such things; to suffer freedom of speech in others,
and to apply myself heartily to philosophy. Him also I must thank for
my hearing first Bacchius, then Tandasis and Marcianus; that I wrote
dialogues in my youth, and took a liking to the philosopher's pallet
and skins, and to the other things which, by the Grecian discipline,
belong to that profession.

7. To Rusticus I owe my first apprehensions that my nature needed
reform and cure; and that I did not fall into the ambition of the
common Sophists, either by composing speculative writings or by
declaiming harangues of exhortation in public; further, that I never
strove to be admired by ostentation of great patience in an ascetic
life, or by display of activity and application; that I gave over the
study of rhetoric, poetry, and the graces of language; and that I did
not pace my house in my senatorial robes, or practise any similar
affectation. I observed also the simplicity of style in his letters,
particularly in that which he wrote to my mother from Sinuessa. I
learned from him to be easily appeased, and to be readily reconciled
with those who had displeased me or given cause of offence, so soon as
they inclined to make their peace; to read with care; not to rest
satisfied with a slight and superficial knowledge; nor quickly to
assent to great talkers. I have him to thank that I met with the
discourses of Epictetus, which he furnished me from his own library.

8. From Apollonius I learned true liberty, and tenacity of purpose; to
regard nothing else, even in the smallest degree, but reason always;
and always to remain unaltered in the agonies of pain, in the losses
of children, or in long diseases. He afforded me a living example of
how the same man can, upon occasion, be most yielding and most
inflexible. He was patient in exposition; and, as might well be seen,
esteemed his fine skill and ability in teaching others the principles
of philosophy as the least of his endowments. It was from him that I
learned how to receive from friends what are thought favours without
seeming humbled by the giver or insensible to the gift.`
