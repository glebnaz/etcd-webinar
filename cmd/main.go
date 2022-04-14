package main

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	cliv3 "go.etcd.io/etcd/client/v3"
)

var (
	upgrader = websocket.Upgrader{}
)

func main() {
	e := echo.New()

	cli, err := cliv3.New(cliv3.Config{
		Endpoints: []string{"localhost:2379"},
		Username:  "gleb-naz",
	})
	if err != nil {
		panic(err)
	}

	route(e, cli)

	e.Start(":9999")
}

func route(e *echo.Echo, cli *cliv3.Client) {
	e.Static("/", "/Users/gleb.nazemnov/Documents/rebrain/etcd/public")
	e.GET("/get", func(c echo.Context) error {
		//todo get
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.POST("/put", func(c echo.Context) error {
		//todo put
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.DELETE("/delete", func(c echo.Context) error {
		//todo delete
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.GET("/watch", func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer ws.Close()

		for {
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
			if err != nil {
				c.Logger().Error(err)
			}

			time.Sleep(2 * time.Second)
		}
	})
}
