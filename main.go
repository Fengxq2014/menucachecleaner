package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	masterName    string
	sentinelAddrs string
	db            int
	password      string
	addr          string
	regex         string
	httpTimeOut   int
	redisCli      *redis.Client
)

func main() {

	flag.StringVar(&masterName, "masterName", "mymaster", "master name")
	flag.StringVar(&sentinelAddrs, "sentinels", "", "sentinels address separate with commas")
	flag.IntVar(&db, "db", 0, "db")
	flag.StringVar(&password, "password", "", "redis password")
	flag.StringVar(&addr, "addr", ":80", "http server to listen,eg:127.0.0.1:80,:80")
	flag.StringVar(&regex, "regex", "JSABCWeiXin-menuKey*", "key search regex")
	flag.IntVar(&httpTimeOut, "timeOut", 15, "read/write time out second")

	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		return
	}

	redisCli = redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: strings.Split(sentinelAddrs, ","),
		DB:            db,
		Password:      password,
	})

	err := redisCli.Ping().Err()
	if err != nil {
		log.Fatal(err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/delete", serveHTTP)

	srv := &http.Server{
		Addr:         addr,
		WriteTimeout: time.Second * time.Duration(httpTimeOut),
		ReadTimeout:  time.Second * time.Duration(httpTimeOut),
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		log.Printf("server listen on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	srv.Shutdown(ctx)
	redisCli.Exists()
	redisCli.Close()
	log.Println("shutting down")
	os.Exit(0)
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	keys, err := redisCli.Keys(regex).Result()
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}
	err = redisCli.Del(keys...).Err()
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}
	fmt.Fprint(w, "ok")
	return
}
