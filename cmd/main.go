package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cliv3 "go.etcd.io/etcd/client/v3"

	"net/http"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Разрешаем подключения с любого источника
		},
	}
)

func main() {
	e := echo.New()

	// Добавляем middleware для CORS
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

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

type Pet struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	ID   string `json:"id"`
}

func route(e *echo.Echo, cli *cliv3.Client) {
	e.Static("/", "/Users/gleb.nazemnov/Documents/rebrain/etcd/public")
	e.GET("/get", func(c echo.Context) error {
		id := c.QueryParams().Get("id")

		if id == "" {
			return c.JSON(http.StatusBadRequest, "id is empty")
		}

		rsp, err := cli.Get(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		if rsp.Count == 0 {
			return c.JSON(http.StatusNotFound, "not found")
		}

		var pet Pet
		err = json.Unmarshal(rsp.Kvs[0].Value, &pet)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		return c.JSON(http.StatusOK, pet)
	})

	e.POST("/put", func(c echo.Context) error {
		var pet Pet

		err := c.Bind(&pet)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		// Получаем текущее значение перед записью
		getRsp, err := cli.Get(c.Request().Context(), pet.ID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		if getRsp.Count > 0 {
			fmt.Printf("prev: %v\n", string(getRsp.Kvs[0].Value))
		} else {
			fmt.Printf("prev: <nil> (new key)\n")
		}

		bytePet, err := json.Marshal(pet)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		_, err = cli.Put(c.Request().Context(), pet.ID, string(bytePet))
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		return c.JSON(http.StatusOK, pet)
	})

	e.DELETE("/delete", func(c echo.Context) error {
		id := c.QueryParams().Get("id")

		if id == "" {
			return c.JSON(http.StatusBadRequest, "id is empty")
		}

		rsp, err := cli.Delete(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		if len(rsp.PrevKvs) > 0 {
			fmt.Printf("prev: %v\n", string(rsp.PrevKvs[0].Value))
		} else {
			fmt.Printf("prev: <nil> (key did not exist)\n")
		}

		return c.String(http.StatusOK, "ok")
	})

	e.GET("/watch", func(c echo.Context) error {
		id := "pet-"

		ch := cli.Watch(c.Request().Context(), id, cliv3.WithPrefix())

		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer ws.Close()

		// Отправляем сообщение клиенту при успешном подключении
		err = ws.WriteMessage(websocket.TextMessage, []byte("Connected to WebSocket server"))
		if err != nil {
			fmt.Println("Error sending initial message:", err)
			return err
		}

		for {
			select {
			case v := <-ch:
				value := v.Events[0].Kv.Value

				var pet Pet
				err = json.Unmarshal(value, &pet)
				if err != nil {
					err = ws.WriteMessage(websocket.TextMessage, []byte("pet is bad"))
					if err != nil {
						fmt.Println(err)
						continue
					}
				}

				err = ws.WriteMessage(websocket.TextMessage, value)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}
	})
}
