package router

import (
	"context"
	"errors"
	"lintang/coba_osm/alg"
	"lintang/coba_osm/domain"
	"lintang/coba_osm/util"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type NavigationService interface {
	ShortestPathETA(ctx context.Context, srcLat, srcLon float64,
		dstLat float64, dstLon float64) (string, float64, []alg.Navigation, bool, []alg.Coordinate, float64, error)
}

type NavigationHandler struct {
	svc NavigationService
}

func NavigatorRouter(r *chi.Mux, svc NavigationService) {
	handler := &NavigationHandler{svc}

	r.Group(func(r chi.Router) {
		r.Route("/api/navigations", func(r chi.Router) {
			r.Post("/shortestPath", handler.shortestPathETA)

		})
	})
}

type SortestPathRequest struct {
	SrcLat float64 `json:"src_lat"`
	SrcLon float64 `json:"src_lon"`
	DstLat float64 `json:"dst_lat"`
	DstLon float64 `json:"dst_lon"`
}

func (s *SortestPathRequest) Bind(r *http.Request) error {
	if s.SrcLat == 0 || s.SrcLon == 0 || s.DstLat == 0 || s.DstLon == 0 {
		return errors.New("invalid request")
	}
	return nil
}

type ShortestPathResponse struct {
	Path        string           `json:"path"`
	Dist        float64          `json:"distance"`
	ETA         float64          `json:"ETA"`
	Navigations []alg.Navigation `json:"navigations"`
	Found       bool             `json:"found"`
	// Route       []alg.Coordinate `json:"route"`
}

func NewShortestPathResponse(path string, distance float64, navs []alg.Navigation, eta float64, route []alg.Coordinate, found bool) *ShortestPathResponse {

	return &ShortestPathResponse{
		Path:        path,
		Dist:        util.RoundFloat(distance, 2),
		ETA:         util.RoundFloat(eta, 2),
		Navigations: navs,
		Found:       found,
		// Route: route,
	}
}

func (h *NavigationHandler) shortestPathETA(w http.ResponseWriter, r *http.Request) {
	data := &SortestPathRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	p, dist, n, found, route, eta, err := h.svc.ShortestPathETA(r.Context(), data.SrcLat, data.SrcLon, data.DstLat, data.DstLon)
	if err != nil {
		if !found {
			render.Render(w, r, ErrInvalidRequest(errors.New("node not found")))
			return
		}
		render.Render(w, r, ErrInternalServerErrorRend(errors.New("internal server error")))
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, NewShortestPathResponse(p, dist, n, eta, route, found))
}

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

func ErrInternalServerErrorRend(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Internal server error.",
		ErrorText:      err.Error(),
	}
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

func ErrChi(err error) render.Renderer {
	statusText := ""
	switch getStatusCode(err) {
	case http.StatusNotFound:
		statusText = "Resource not found."
	case http.StatusInternalServerError:
		statusText = "Internal server error."
	case http.StatusConflict:
		statusText = "Resource conflict."
	case http.StatusBadRequest:
		statusText = "Bad request."
	default:
		statusText = "Error."
	}

	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: getStatusCode(err),
		StatusText:     statusText,
		ErrorText:      err.Error(),
	}
}

func getStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var ierr *domain.Error
	if !errors.As(err, &ierr) {
		return http.StatusInternalServerError
	} else {
		switch ierr.Code() {
		case domain.ErrInternalServerError:
			return http.StatusInternalServerError
		case domain.ErrNotFound:
			return http.StatusNotFound
		case domain.ErrConflict:
			return http.StatusConflict
		case domain.ErrBadParamInput:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}

}
