package opds

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vanadium23/kompanion/internal/auth"
	"github.com/vanadium23/kompanion/internal/library"
	"github.com/vanadium23/kompanion/internal/sync"
	"github.com/vanadium23/kompanion/pkg/logger"
)

type OPDSRouter struct {
	urlPrefix string
	books     library.Shelf
	logger    logger.Interface
}

func NewRouter(
	handler *gin.RouterGroup,
	l logger.Interface,
	a auth.AuthInterface,
	p sync.Progress,
	shelf library.Shelf) {
	urlPrefix := strings.TrimSuffix(handler.BasePath(), "/")
	sh := &OPDSRouter{urlPrefix, shelf, l}

	h := handler.Group("/opds")
	h.Use(basicAuth(a))
	{
		h.GET("/", sh.listShelves)
		h.GET("/newest/", sh.listNewest)
		h.GET("/book/:bookID/download", sh.downloadBook)
		// TODO: search
	}
}

func (r *OPDSRouter) listShelves(c *gin.Context) {
	shelves := []Entry{
		{
			ID:      "urn:kompanion:newest",
			Updated: time.Now().UTC().Format(AtomTime),
			Title:   "By Newest",
			Link: []Link{
				{
					Href: r.urlPrefix + "/opds/newest/",
					Type: "application/atom+xml;type=feed;profile=opds-catalog",
				},
			},
		},
	}
	links := []Link{}
	feed := BuildFeed("urn:kompanion:main", "KOmpanion library", r.urlPrefix+"/opds", shelves, links, r.urlPrefix)
	c.XML(http.StatusOK, feed)
}

func (r *OPDSRouter) listNewest(c *gin.Context) {
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	books, err := r.books.ListBooks(c.Request.Context(), "created_at", "desc", page, 10)
	if err != nil {
		r.logger.Error("failed to list newest books", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error", "code": 1001})
		return
	}
	baseUrl := r.urlPrefix + "/opds/newest/"
	entries := translateBooksToEntries(books.Books, r.urlPrefix)
	navLinks := formNavLinks(baseUrl, books)
	feed := BuildFeed("urn:kompanion:newest", "KOmpanion library", baseUrl, entries, navLinks, r.urlPrefix)
	c.XML(http.StatusOK, feed)
}

func (r *OPDSRouter) downloadBook(c *gin.Context) {
	bookID := c.Param("bookID")

	book, file, err := r.books.DownloadBook(c.Request.Context(), bookID)
	if err != nil {
		r.logger.Error(err, "http - v1 - shelf - downloadBook")
		c.JSON(500, gin.H{"message": "internal server error"})
		return
	}
	defer file.Close()

	c.Header("Content-Disposition", "attachment; filename="+book.Filename())
	c.Header("Content-Type", "application/octet-stream")
	c.File(file.Name())
}

func basicAuth(auth auth.AuthInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="KOmpanion OPDS"`)
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "code": 2001})
			c.Abort()
			return
		}
		if !auth.CheckDevicePassword(c.Request.Context(), username, password, true) {
			if !auth.CheckPassword(c.Request.Context(), username, password) {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "code": 2001})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
