package crud

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func (e *ErrResponse) Error() string {
	return fmt.Sprintf("%d %s %s", e.AppCode, e.StatusText, e.ErrorText)
}

func ErrBadRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "Internal error.",
		ErrorText:      err.Error(),
	}
}

func ErrForbiddenText(txt string) render.Renderer {
	return &ErrResponse{
		HTTPStatusCode: http.StatusForbidden,
		StatusText:     txt,
	}
}

var ErrNotFound = &ErrResponse{
	HTTPStatusCode: http.StatusNotFound,
	StatusText:     "Resource not found.",
}

var ErrUnauthorized = &ErrResponse{
	HTTPStatusCode: http.StatusUnauthorized,
	StatusText:     "Unauthorized.",
}

var ErrForbidden = &ErrResponse{
	HTTPStatusCode: http.StatusForbidden,
	StatusText:     "Forbidden.",
}
