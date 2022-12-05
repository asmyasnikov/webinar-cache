package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/asmyasnikov/webinar-cache/internal/storage"
)

var errNotEnthoughtFreeSeats = errors.New("not enough free seats")

type server struct {
	mux   http.ServeMux
	store storage.Storage
}

func New(store storage.Storage) *server {
	s := &server{
		store: store,
	}

	s.mux.HandleFunc("/", s.IndexPageHandler)
	s.mux.HandleFunc("/get/", s.GetFreeSeatsHandler)
	s.mux.HandleFunc("/buy/", s.BuyTicketHandler)

	return s
}

func (s *server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mux.ServeHTTP(writer, request)
}

func (s *server) GetFreeSeatsHandler(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	id := path.Base(request.URL.Path)

	start := time.Now()
	freeSeats, err := s.store.FreeSeats(ctx, id)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	duration := time.Since(start)
	s.writeAnswer(writer, freeSeats, duration)
}

func (s *server) BuyTicketHandler(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	id := path.Base(request.URL.Path)

	start := time.Now()
	freeSeats, err := s.store.DecrementFreeSeats(ctx, id)
	if err != nil {
		if errors.Is(err, errNotEnthoughtFreeSeats) {
			http.Error(writer, "Not enough free seats", http.StatusPreconditionFailed)
		} else {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	duration := time.Since(start)
	s.writeAnswer(writer, freeSeats, duration)
}

func (s *server) writeAnswer(writer http.ResponseWriter, freeSeats int64, duration time.Duration) {
	_, _ = fmt.Fprintf(writer, "%v\n\nDuration: %v\n", freeSeats, duration)
}

func (s *server) IndexPageHandler(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	var busIDs []string

	busIDs, err := s.store.BusIDs(ctx)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/html")
	writer.WriteHeader(http.StatusOK)

	_, _ = io.WriteString(writer, `Bus table<br />
<br />
<table border="1">
	<tr>
		<th>ID</th>
		<th>Get free seats link</th>
		<th>Buy ticket link</th>
	</tr>
`)
	for _, id := range busIDs {
		_, _ = fmt.Fprintf(writer, `<tr>
	<td>%v</td>
	<td><a href="/get/%v">/get/%v</a></td>
	<td><a href="/buy/%v">/buy/%v</a></td>
</tr>`, id, id, id, id, id)
	}
	_, _ = io.WriteString(writer, "</table>")
}
