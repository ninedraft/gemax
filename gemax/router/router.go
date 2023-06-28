// Package router implements a simple parametrized router for gemax.
package router

import (
	"context"
	"strings"

	"github.com/ninedraft/gemax/gemax"
)

// IncomingRequest is an interface for gemax.IncomingRequest with additional
// method Param, that returns value of parametrized path part.
// If path part is not parametrized or requested parameter is not exist, Param returns false.
//
// Examples:
//
//	/hello/:name -> /hello/world -> Param("name") -> "world", true
//	/hello/:name -> /hello -> Param("boop") -> "", false
type IncomingRequest interface {
	gemax.IncomingRequest
	Param(name string) (string, bool)
}

type incomingRequest struct {
	gemax.IncomingRequest
	params map[string]string
}

func (req *incomingRequest) Param(name string) (string, bool) {
	if req == nil || req.params == nil {
		return "", false
	}

	val, ok := req.params[name]
	return val, ok
}

// Router is a simple parametrized router for gemax.
// Routes are registered with HandleParams and Handle methods.
// Routes has format similar to http.ServeMux: "/path/:param" or "/path".
type Router struct {
	root    *node
	handers []handler
}

type handler struct {
	pattern    string
	omitParams bool
	fn         func(context.Context, gemax.ResponseWriter, IncomingRequest)
}

// New returns new instance of Router.
func New() *Router {
	return &Router{
		root: newNode(-1),
	}
}

// HandleParamsFn describes function that handles parametrized incoming request.
type HandleParamsFn = func(ctx context.Context, rw gemax.ResponseWriter, req IncomingRequest)

// HandleParams registers new handler for parametrized path.
func (router *Router) HandleParams(pattern string, handle HandleParamsFn) {
	i := len(router.handers)
	router.handers = append(router.handers, handler{
		pattern:    pattern,
		omitParams: !strings.Contains(pattern, ":"),
		fn:         handle,
	})
	router.add(pattern, i)
}

// Handle registers new handler for path.
func (router *Router) Handle(pattern string, handle gemax.Handler) {
	h := func(ctx context.Context, rw gemax.ResponseWriter, req IncomingRequest) {
		handle(ctx, rw, req)
	}

	router.HandleParams(pattern, h)
}

var _ gemax.Handler = new(Router).Serve

// Serve method handles incoming request. It can be used as gemax.Handler.
func (router *Router) Serve(ctx context.Context, rw gemax.ResponseWriter, req gemax.IncomingRequest) {
	params := map[string]string{}
	index := router.get(req.URL().Path, params)

	if index < 0 {
		return
	}

	handler := router.handers[index]

	handler.fn(ctx, rw, &incomingRequest{
		IncomingRequest: req,
		params:          params,
	})
}

type node struct {
	part     string
	children map[string]*node
	paramed  *node
	index    int
	param    string
}

func (router *Router) add(path string, index int) {
	currentNode := router.root

	for path != "" {
		part, rest, _ := strings.Cut(path, "/")
		path = rest

		childNode := currentNode.child(part)
		if childNode != nil {
			currentNode = childNode
			continue
		}

		newNode := newNode(-1)

		param, hasParam := strings.CutPrefix(part, ":")
		switch {
		case hasParam:
			newNode.param = param
			currentNode.paramed = newNode
		default:
			newNode.part = part
			currentNode.children[part] = newNode
		}

		currentNode = newNode
	}

	currentNode.index = index
}

func (router *Router) get(path string, params map[string]string) int {
	currentNode := router.root

	for path != "" {
		part, rest, _ := strings.Cut(path, "/")
		path = rest

		childNode := currentNode.child(part)
		if childNode == nil {
			return -1
		}

		if childNode.param != "" && params != nil {
			params[childNode.param] = part
		}

		currentNode = childNode
	}

	return currentNode.index
}

func (n *node) child(part string) *node {
	child, ok := n.children[part]
	if ok {
		return child
	}

	if n.paramed != nil {
		return n.paramed
	}

	return nil
}

func newNode(index int) *node {
	return &node{
		children: map[string]*node{},
		index:    index,
	}
}
