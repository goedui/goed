//go:build linux

package linux

/*
#cgo LDFLAGS: -ldl
#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

static void *xlib;

int x11_load(void) {
	xlib = dlopen("libX11.so.6", RTLD_NOW | RTLD_GLOBAL);
	return xlib != 0;
}

void *x11_sym(const char *n) { return dlsym(xlib, n); }
char *x11_err(void) { return dlerror(); }

int x11_noop_error(void *dpy, void *ev) { (void)dpy; (void)ev; return 0; }
int x11_io_error(void *dpy) { (void)dpy; return 0; }

void x11_quiet(void *set_error, void *set_io) {
	if (set_error) ((void *(*)(void *))set_error)((void *)x11_noop_error);
	if (set_io) ((void *(*)(void *))set_io)((void *)x11_io_error);
}

void *x11_malloc(size_t n) { return malloc(n); }
void x11_free(void *p) { if (p) free(p); }

typedef struct {
	int width;
	int height;
	int xoffset;
	int format;
	char *data;
} x11_image_head;

void x11_image_clear_data(void *img) {
	if (img) ((x11_image_head *)img)->data = 0;
}

uintptr_t xc0(void *f) { return ((uintptr_t(*)(void))f)(); }
uintptr_t xc1(void *f, uintptr_t a) { return ((uintptr_t(*)(uintptr_t))f)(a); }
uintptr_t xc2(void *f, uintptr_t a, uintptr_t b) { return ((uintptr_t(*)(uintptr_t, uintptr_t))f)(a, b); }
uintptr_t xc3(void *f, uintptr_t a, uintptr_t b, uintptr_t c) { return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t))f)(a, b, c); }
uintptr_t xc4(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d);
}
uintptr_t xc5(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d, uintptr_t e) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d, e);
}
uintptr_t xc6(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d, uintptr_t e, uintptr_t g) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d, e, g);
}
uintptr_t xc7(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d, uintptr_t e, uintptr_t g, uintptr_t h) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d, e, g, h);
}
uintptr_t xc8(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d, uintptr_t e, uintptr_t g, uintptr_t h, uintptr_t i) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d, e, g, h, i);
}
uintptr_t xc9(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d, uintptr_t e, uintptr_t g, uintptr_t h, uintptr_t i, uintptr_t j) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d, e, g, h, i, j);
}
uintptr_t xc10(void *f, uintptr_t a, uintptr_t b, uintptr_t c, uintptr_t d, uintptr_t e, uintptr_t g, uintptr_t h, uintptr_t i, uintptr_t j, uintptr_t k) {
	return ((uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t))f)(a, b, c, d, e, g, h, i, j, k);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type x11api struct {
	XOpenDisplay         unsafe.Pointer
	XCloseDisplay        unsafe.Pointer
	XRootWindow          unsafe.Pointer
	XDisplayWidth        unsafe.Pointer
	XDisplayHeight       unsafe.Pointer
	XDisplayWidthMM      unsafe.Pointer
	XDefaultDepth        unsafe.Pointer
	XDefaultVisual       unsafe.Pointer
	XBlackPixel          unsafe.Pointer
	XWhitePixel          unsafe.Pointer
	XCreateSimpleWindow  unsafe.Pointer
	XSelectInput         unsafe.Pointer
	XMapWindow           unsafe.Pointer
	XDestroyWindow       unsafe.Pointer
	XNextEvent           unsafe.Pointer
	XPending             unsafe.Pointer
	XFlush               unsafe.Pointer
	XInternAtom          unsafe.Pointer
	XChangeProperty      unsafe.Pointer
	XSendEvent           unsafe.Pointer
	XMoveResizeWindow    unsafe.Pointer
	XIconifyWindow       unsafe.Pointer
	XGetWindowAttributes unsafe.Pointer
	XStoreName           unsafe.Pointer
	XSetWMProtocols      unsafe.Pointer
	XCreateGC            unsafe.Pointer
	XFreeGC              unsafe.Pointer
	XPutImage            unsafe.Pointer
	XCreateImage         unsafe.Pointer
	XDestroyImage        unsafe.Pointer
	XLookupString        unsafe.Pointer
	XSetErrorHandler     unsafe.Pointer
	XSetIOErrorHandler   unsafe.Pointer
	XInitThreads         unsafe.Pointer
	XUngrabPointer       unsafe.Pointer
	XConnectionNumber    unsafe.Pointer
	XDefaultScreen       unsafe.Pointer
	XFree                unsafe.Pointer
}

const (
	xKeyPress         = 2
	xKeyRelease       = 3
	xButtonPress      = 4
	xButtonRelease    = 5
	xMotionNotify     = 6
	xEnterNotify      = 7
	xLeaveNotify      = 8
	xFocusIn          = 9
	xFocusOut         = 10
	xExpose           = 12
	xDestroyNotify    = 17
	xUnmapNotify      = 18
	xMapNotify        = 19
	xConfigureNotify  = 22
	xClientMessage    = 33

	xKeyPressMask         = 1 << 0
	xKeyReleaseMask       = 1 << 1
	xButtonPressMask      = 1 << 2
	xButtonReleaseMask    = 1 << 3
	xPointerMotionMask    = 1 << 6
	xExposureMask         = 1 << 15
	xStructureNotifyMask  = 1 << 17
	xSubstructureNotify   = 1 << 19
	xSubstructureRedirect = 1 << 20
	xFocusChangeMask      = 1 << 21

	xPropModeReplace = 0
	xaAtom           = 4
	xaCardinal       = 6

	zPixmap = 2
)

var x x11api

func loadX11() error {
	C.x11_err()
	if C.x11_load() == 0 {
		err := C.GoString(C.x11_err())
		if err == "" {
			err = "dlopen 失败"
		}
		return fmt.Errorf("github.com/goedui/goed/ui: 无法加载 libX11.so.6（请安装 libx11-6）: %s", err)
	}
	need := []struct {
		name string
		dst  *unsafe.Pointer
	}{
		{"XOpenDisplay", &x.XOpenDisplay},
		{"XCloseDisplay", &x.XCloseDisplay},
		{"XRootWindow", &x.XRootWindow},
		{"XDisplayWidth", &x.XDisplayWidth},
		{"XDisplayHeight", &x.XDisplayHeight},
		{"XDisplayWidthMM", &x.XDisplayWidthMM},
		{"XDefaultDepth", &x.XDefaultDepth},
		{"XDefaultVisual", &x.XDefaultVisual},
		{"XBlackPixel", &x.XBlackPixel},
		{"XWhitePixel", &x.XWhitePixel},
		{"XCreateSimpleWindow", &x.XCreateSimpleWindow},
		{"XSelectInput", &x.XSelectInput},
		{"XMapWindow", &x.XMapWindow},
		{"XDestroyWindow", &x.XDestroyWindow},
		{"XNextEvent", &x.XNextEvent},
		{"XPending", &x.XPending},
		{"XFlush", &x.XFlush},
		{"XInternAtom", &x.XInternAtom},
		{"XChangeProperty", &x.XChangeProperty},
		{"XSendEvent", &x.XSendEvent},
		{"XMoveResizeWindow", &x.XMoveResizeWindow},
		{"XIconifyWindow", &x.XIconifyWindow},
		{"XGetWindowAttributes", &x.XGetWindowAttributes},
		{"XStoreName", &x.XStoreName},
		{"XSetWMProtocols", &x.XSetWMProtocols},
		{"XCreateGC", &x.XCreateGC},
		{"XFreeGC", &x.XFreeGC},
		{"XPutImage", &x.XPutImage},
		{"XCreateImage", &x.XCreateImage},
		{"XLookupString", &x.XLookupString},
		{"XUngrabPointer", &x.XUngrabPointer},
		{"XDefaultScreen", &x.XDefaultScreen},
	}
	for _, s := range need {
		p := xsym(s.name)
		if p == nil {
			return fmt.Errorf("github.com/goedui/goed/ui: libX11 缺少符号 %s", s.name)
		}
		*s.dst = p
	}
	x.XDestroyImage = xsym("XDestroyImage")
	x.XSetErrorHandler = xsym("XSetErrorHandler")
	x.XSetIOErrorHandler = xsym("XSetIOErrorHandler")
	x.XInitThreads = xsym("XInitThreads")
	x.XConnectionNumber = xsym("XConnectionNumber")
	x.XFree = xsym("XFree")
	return nil
}

func xsym(name string) unsafe.Pointer {
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	return C.x11_sym(c)
}

func xc0(f unsafe.Pointer) uintptr {
	return uintptr(C.xc0(f))
}
func xc1(f unsafe.Pointer, a uintptr) uintptr {
	return uintptr(C.xc1(f, C.uintptr_t(a)))
}
func xc2(f unsafe.Pointer, a, b uintptr) uintptr {
	return uintptr(C.xc2(f, C.uintptr_t(a), C.uintptr_t(b)))
}
func xc3(f unsafe.Pointer, a, b, c uintptr) uintptr {
	return uintptr(C.xc3(f, C.uintptr_t(a), C.uintptr_t(b), C.uintptr_t(c)))
}
func xc4(f unsafe.Pointer, a, b, c, d uintptr) uintptr {
	return uintptr(C.xc4(f, C.uintptr_t(a), C.uintptr_t(b), C.uintptr_t(c), C.uintptr_t(d)))
}
func xc5(f unsafe.Pointer, a, b, c, d, e uintptr) uintptr {
	return uintptr(C.xc5(f, C.uintptr_t(a), C.uintptr_t(b), C.uintptr_t(c), C.uintptr_t(d), C.uintptr_t(e)))
}
func xc8(f unsafe.Pointer, a, b, c, d, e, g, h, i uintptr) uintptr {
	return uintptr(C.xc8(f, C.uintptr_t(a), C.uintptr_t(b), C.uintptr_t(c), C.uintptr_t(d), C.uintptr_t(e), C.uintptr_t(g), C.uintptr_t(h), C.uintptr_t(i)))
}
func xc9(f unsafe.Pointer, a, b, c, d, e, g, h, i, j uintptr) uintptr {
	return uintptr(C.xc9(f, C.uintptr_t(a), C.uintptr_t(b), C.uintptr_t(c), C.uintptr_t(d), C.uintptr_t(e), C.uintptr_t(g), C.uintptr_t(h), C.uintptr_t(i), C.uintptr_t(j)))
}
func xc10(f unsafe.Pointer, a, b, c, d, e, g, h, i, j, k uintptr) uintptr {
	return uintptr(C.xc10(f, C.uintptr_t(a), C.uintptr_t(b), C.uintptr_t(c), C.uintptr_t(d), C.uintptr_t(e), C.uintptr_t(g), C.uintptr_t(h), C.uintptr_t(i), C.uintptr_t(j), C.uintptr_t(k)))
}

func internAtom(dpy uintptr, name string, onlyIfExists bool) uintptr {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	exists := uintptr(0)
	if onlyIfExists {
		exists = 1
	}
	return xc3(x.XInternAtom, dpy, uintptr(unsafe.Pointer(cname)), exists)
}

func cfree(p *C.char) { C.free(unsafe.Pointer(p)) }
