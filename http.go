package main

import (
	"context"
	"fmt"
	"github.com/gravataLonga/ninja/evaluator"
	. "github.com/gravataLonga/ninja/object"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type HttpObject struct {
	settings *Hash
}

func (h *HttpObject) Type() ObjectType {
	return "HTTP"
}

func (h *HttpObject) Inspect() string {
	return "plugin<http>"
}

func NewHttp(args ...Object) Object {
	err := Check("args", args, ExactArgs(1), WithTypes(HASH_OBJ))

	if err != nil {
		return NewError(err.Error())
	}

	return &HttpObject{
		args[0].(*Hash),
	}
}

func (h *HttpObject) Call(method string, args ...Object) Object {
	switch method {
	case "listen":
		return h.Listen(args...)
	case "handle":
		return h.Handle(args...)
	}
	return NewErrorFormat("method %s not exists", method)
}

func (h *HttpObject) Listen(args ...Object) Object {
	portStr := &String{Value: "port"}
	port := "8080"
	portHash, ok := h.settings.Pairs[portStr.HashKey()]
	if ok {
		portHashValue := portHash.Value
		if intPort, ok := portHashValue.(*Integer); ok {
			port = intPort.Inspect()
		}
		if strPort, ok := portHashValue.(*String); ok {
			port = strPort.Inspect()
		}
	}

	server := &http.Server{
		Addr: ":" + port,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.SetKeepAlivesEnabled(false)
		close(done)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return NewError(fmt.Sprintf("Could not listen on %s: %v", port, err))
	}

	<-done
	return nil
}

// Handle take two argument, first one is path string, second is function to handle request.
func (h *HttpObject) Handle(args ...Object) Object {
	err := Check("args", args, ExactArgs(2), WithTypes(STRING_OBJ, FUNCTION_OBJ))
	if err != nil {
		return NewError(err.Error())
	}

	path := args[0].(*String).Value
	fn := args[1].(*FunctionLiteral)

	http.HandleFunc(path, func(writer http.ResponseWriter, request *http.Request) {
		evaluator.Eval(fn.Body, NewEnvironment())
	})

	return nil
}
