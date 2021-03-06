// SPDX-License-Identifier: Unlicense OR MIT

package gl

import (
	"strings"
	"syscall/js"
)

type Functions struct {
	Ctx                             js.Value
	EXT_disjoint_timer_query        js.Value
	EXT_disjoint_timer_query_webgl2 js.Value

	// Cached JS arrays.
	byteBuf  js.Value
	int32Buf js.Value
}

func (f *Functions) Init() {
	f.EXT_disjoint_timer_query_webgl2 = f.getExtension("EXT_disjoint_timer_query_webgl2")
	if f.EXT_disjoint_timer_query_webgl2 == js.Null() {
		f.EXT_disjoint_timer_query = f.getExtension("EXT_disjoint_timer_query")
	}
	// Enable extensions.
	f.getExtension("OES_texture_half_float")
	f.getExtension("OES_texture_float")
	f.getExtension("EXT_sRGB")
	// WebGL2 extensions
	f.getExtension("EXT_color_buffer_half_float")
	f.getExtension("EXT_color_buffer_float")
}

func (f *Functions) getExtension(name string) js.Value {
	return f.Ctx.Call("getExtension", name)
}

func (f *Functions) ActiveTexture(t Enum) {
	f.Ctx.Call("activeTexture", int(t))
}
func (f *Functions) AttachShader(p Program, s Shader) {
	f.Ctx.Call("attachShader", js.Value(p), js.Value(s))
}
func (f *Functions) BeginQuery(target Enum, query Query) {
	if f.EXT_disjoint_timer_query_webgl2 != js.Null() {
		f.Ctx.Call("beginQuery", int(target), js.Value(query))
	} else {
		f.EXT_disjoint_timer_query.Call("beginQueryEXT", int(target), js.Value(query))
	}
}
func (f *Functions) BindAttribLocation(p Program, a Attrib, name string) {
	f.Ctx.Call("bindAttribLocation", js.Value(p), int(a), name)
}
func (f *Functions) BindBuffer(target Enum, b Buffer) {
	f.Ctx.Call("bindBuffer", int(target), js.Value(b))
}
func (f *Functions) BindFramebuffer(target Enum, fb Framebuffer) {
	f.Ctx.Call("bindFramebuffer", int(target), js.Value(fb))
}
func (f *Functions) BindRenderbuffer(target Enum, rb Renderbuffer) {
	f.Ctx.Call("bindRenderbuffer", int(target), js.Value(rb))
}
func (f *Functions) BindTexture(target Enum, t Texture) {
	f.Ctx.Call("bindTexture", int(target), js.Value(t))
}
func (f *Functions) BlendEquation(mode Enum) {
	f.Ctx.Call("blendEquation", int(mode))
}
func (f *Functions) BlendFunc(sfactor, dfactor Enum) {
	f.Ctx.Call("blendFunc", int(sfactor), int(dfactor))
}
func (f *Functions) BufferData(target Enum, src []byte, usage Enum) {
	f.Ctx.Call("bufferData", int(target), f.byteArrayOf(src), int(usage))
}
func (f *Functions) CheckFramebufferStatus(target Enum) Enum {
	return Enum(f.Ctx.Call("checkFramebufferStatus", int(target)).Int())
}
func (f *Functions) Clear(mask Enum) {
	f.Ctx.Call("clear", int(mask))
}
func (f *Functions) ClearColor(red, green, blue, alpha float32) {
	f.Ctx.Call("clearColor", red, green, blue, alpha)
}
func (f *Functions) ClearDepthf(d float32) {
	f.Ctx.Call("clearDepth", d)
}
func (f *Functions) CompileShader(s Shader) {
	f.Ctx.Call("compileShader", js.Value(s))
}
func (f *Functions) CreateBuffer() Buffer {
	return Buffer(f.Ctx.Call("createBuffer"))
}
func (f *Functions) CreateFramebuffer() Framebuffer {
	return Framebuffer(f.Ctx.Call("createFramebuffer"))
}
func (f *Functions) CreateProgram() Program {
	return Program(f.Ctx.Call("createProgram"))
}
func (f *Functions) CreateQuery() Query {
	return Query(f.Ctx.Call("createQuery"))
}
func (f *Functions) CreateRenderbuffer() Renderbuffer {
	return Renderbuffer(f.Ctx.Call("createRenderbuffer"))
}
func (f *Functions) CreateShader(ty Enum) Shader {
	return Shader(f.Ctx.Call("createShader", int(ty)))
}
func (f *Functions) CreateTexture() Texture {
	return Texture(f.Ctx.Call("createTexture"))
}
func (f *Functions) DeleteBuffer(v Buffer) {
	f.Ctx.Call("deleteBuffer", js.Value(v))
}
func (f *Functions) DeleteFramebuffer(v Framebuffer) {
	f.Ctx.Call("deleteFramebuffer", js.Value(v))
}
func (f *Functions) DeleteProgram(p Program) {
	f.Ctx.Call("deleteProgram", js.Value(p))
}
func (f *Functions) DeleteQuery(query Query) {
	if f.EXT_disjoint_timer_query_webgl2 != js.Null() {
		f.Ctx.Call("deleteQuery", js.Value(query))
	} else {
		f.EXT_disjoint_timer_query.Call("deleteQueryEXT", js.Value(query))
	}
}
func (f *Functions) DeleteShader(s Shader) {
	f.Ctx.Call("deleteShader", js.Value(s))
}
func (f *Functions) DeleteRenderbuffer(v Renderbuffer) {
	f.Ctx.Call("deleteRenderbuffer", js.Value(v))
}
func (f *Functions) DeleteTexture(v Texture) {
	f.Ctx.Call("deleteTexture", js.Value(v))
}
func (f *Functions) DepthFunc(fn Enum) {
	f.Ctx.Call("depthFunc", int(fn))
}
func (f *Functions) DepthMask(mask bool) {
	f.Ctx.Call("depthMask", mask)
}
func (f *Functions) DisableVertexAttribArray(a Attrib) {
	f.Ctx.Call("disableVertexAttribArray", int(a))
}
func (f *Functions) Disable(cap Enum) {
	f.Ctx.Call("disable", int(cap))
}
func (f *Functions) DrawArrays(mode Enum, first, count int) {
	f.Ctx.Call("drawArrays", int(mode), first, count)
}
func (f *Functions) DrawElements(mode Enum, count int, ty Enum, offset int) {
	f.Ctx.Call("drawElements", int(mode), count, int(ty), offset)
}
func (f *Functions) Enable(cap Enum) {
	f.Ctx.Call("enable", int(cap))
}
func (f *Functions) EnableVertexAttribArray(a Attrib) {
	f.Ctx.Call("enableVertexAttribArray", int(a))
}
func (f *Functions) EndQuery(target Enum) {
	if f.EXT_disjoint_timer_query_webgl2 != js.Null() {
		f.Ctx.Call("endQuery", int(target))
	} else {
		f.EXT_disjoint_timer_query.Call("endQueryEXT", int(target))
	}
}
func (f *Functions) Finish() {
	f.Ctx.Call("finish")
}
func (f *Functions) FramebufferRenderbuffer(target, attachment, renderbuffertarget Enum, renderbuffer Renderbuffer) {
	f.Ctx.Call("framebufferRenderbuffer", int(target), int(attachment), int(renderbuffertarget), js.Value(renderbuffer))
}
func (f *Functions) FramebufferTexture2D(target, attachment, texTarget Enum, t Texture, level int) {
	f.Ctx.Call("framebufferTexture2D", int(target), int(attachment), int(texTarget), js.Value(t), level)
}
func (f *Functions) GetError() Enum {
	return Enum(f.Ctx.Call("getError").Int())
}
func (f *Functions) GetRenderbufferParameteri(target, pname Enum) int {
	return paramVal(f.Ctx.Call("getRenderbufferParameteri", int(pname)))
}
func (f *Functions) GetFramebufferAttachmentParameteri(target, attachment, pname Enum) int {
	return paramVal(f.Ctx.Call("getFramebufferAttachmentParameter", int(target), int(attachment), int(pname)))
}
func (f *Functions) GetBinding(pname Enum) Object {
	return Object(f.Ctx.Call("getParameter", int(pname)))
}
func (f *Functions) GetInteger(pname Enum) int {
	return paramVal(f.Ctx.Call("getParameter", int(pname)))
}
func (f *Functions) GetProgrami(p Program, pname Enum) int {
	return paramVal(f.Ctx.Call("getProgramParameter", js.Value(p), int(pname)))
}
func (f *Functions) GetProgramInfoLog(p Program) string {
	return f.Ctx.Call("getProgramInfoLog", js.Value(p)).String()
}
func (f *Functions) GetQueryObjectuiv(query Query, pname Enum) uint {
	if f.EXT_disjoint_timer_query_webgl2 != js.Null() {
		return uint(paramVal(f.Ctx.Call("getQueryParameter", js.Value(query), int(pname))))
	} else {
		return uint(paramVal(f.EXT_disjoint_timer_query.Call("getQueryObjectEXT", js.Value(query), int(pname))))
	}
}
func (f *Functions) GetShaderi(s Shader, pname Enum) int {
	return paramVal(f.Ctx.Call("getShaderParameter", js.Value(s), int(pname)))
}
func (f *Functions) GetShaderInfoLog(s Shader) string {
	return f.Ctx.Call("getShaderInfoLog", js.Value(s)).String()
}
func (f *Functions) GetString(pname Enum) string {
	switch pname {
	case EXTENSIONS:
		extsjs := f.Ctx.Call("getSupportedExtensions")
		var exts []string
		for i := 0; i < extsjs.Length(); i++ {
			exts = append(exts, "GL_"+extsjs.Index(i).String())
		}
		return strings.Join(exts, " ")
	default:
		return f.Ctx.Call("getParameter", int(pname)).String()
	}
}
func (f *Functions) GetUniformLocation(p Program, name string) Uniform {
	return Uniform(f.Ctx.Call("getUniformLocation", js.Value(p), name))
}
func (f *Functions) InvalidateFramebuffer(target, attachment Enum) {
	fn := f.Ctx.Get("invalidateFramebuffer")
	if fn != js.Undefined() {
		if f.int32Buf == (js.Value{}) {
			f.int32Buf = js.Global().Get("Int32Array").New(1)
		}
		f.int32Buf.SetIndex(0, int32(attachment))
		f.Ctx.Call("invalidateFramebuffer", int(target), f.int32Buf)
	}
}
func (f *Functions) LinkProgram(p Program) {
	f.Ctx.Call("linkProgram", js.Value(p))
}
func (f *Functions) PixelStorei(pname Enum, param int32) {
	f.Ctx.Call("pixelStorei", int(pname), param)
}
func (f *Functions) RenderbufferStorage(target, internalformat Enum, width, height int) {
	f.Ctx.Call("renderbufferStorage", int(target), int(internalformat), width, height)
}
func (f *Functions) ReadPixels(x, y, width, height int, format, ty Enum, data []byte) {
	f.resizeByteBuffer(len(data))
	f.Ctx.Call("readPixels", x, y, width, height, int(format), int(ty), f.byteBuf)
	js.CopyBytesToGo(data, f.byteBuf)
}
func (f *Functions) Scissor(x, y, width, height int32) {
	f.Ctx.Call("scissor", x, y, width, height)
}
func (f *Functions) ShaderSource(s Shader, src string) {
	f.Ctx.Call("shaderSource", js.Value(s), src)
}
func (f *Functions) TexImage2D(target Enum, level int, internalFormat int, width, height int, format, ty Enum, data []byte) {
	f.Ctx.Call("texImage2D", int(target), int(level), int(internalFormat), int(width), int(height), 0, int(format), int(ty), f.byteArrayOf(data))
}
func (f *Functions) TexSubImage2D(target Enum, level int, x, y, width, height int, format, ty Enum, data []byte) {
	f.Ctx.Call("texSubImage2D", int(target), level, x, y, width, height, int(format), int(ty), f.byteArrayOf(data))
}
func (f *Functions) TexParameteri(target, pname Enum, param int) {
	f.Ctx.Call("texParameteri", int(target), int(pname), int(param))
}
func (f *Functions) Uniform1f(dst Uniform, v float32) {
	f.Ctx.Call("uniform1f", js.Value(dst), v)
}
func (f *Functions) Uniform1i(dst Uniform, v int) {
	f.Ctx.Call("uniform1i", js.Value(dst), v)
}
func (f *Functions) Uniform2f(dst Uniform, v0, v1 float32) {
	f.Ctx.Call("uniform2f", js.Value(dst), v0, v1)
}
func (f *Functions) Uniform3f(dst Uniform, v0, v1, v2 float32) {
	f.Ctx.Call("uniform3f", js.Value(dst), v0, v1, v2)
}
func (f *Functions) Uniform4f(dst Uniform, v0, v1, v2, v3 float32) {
	f.Ctx.Call("uniform4f", js.Value(dst), v0, v1, v2, v3)
}
func (f *Functions) UseProgram(p Program) {
	f.Ctx.Call("useProgram", js.Value(p))
}
func (f *Functions) VertexAttribPointer(dst Attrib, size int, ty Enum, normalized bool, stride, offset int) {
	f.Ctx.Call("vertexAttribPointer", int(dst), size, int(ty), normalized, stride, offset)
}
func (f *Functions) Viewport(x, y, width, height int) {
	f.Ctx.Call("viewport", x, y, width, height)
}

func (f *Functions) byteArrayOf(data []byte) js.Value {
	if len(data) == 0 {
		return js.Null()
	}
	f.resizeByteBuffer(len(data))
	js.CopyBytesToJS(f.byteBuf, data)
	return f.byteBuf
}

func (f *Functions) resizeByteBuffer(n int) {
	if n == 0 {
		return
	}
	if f.byteBuf != (js.Value{}) && f.byteBuf.Length() >= n {
		return
	}
	f.byteBuf = js.Global().Get("Uint8Array").New(n)
}

func paramVal(v js.Value) int {
	switch v.Type() {
	case js.TypeBoolean:
		if b := v.Bool(); b {
			return 1
		} else {
			return 0
		}
	case js.TypeNumber:
		return v.Int()
	default:
		panic("unknown parameter type")
	}
}
