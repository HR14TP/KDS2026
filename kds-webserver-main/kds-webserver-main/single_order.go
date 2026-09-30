package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func getSingleOrderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderid := vars["id"]
	order := getDatabaseOrder(config, orderid)
	val, _ := json.Marshal(order)
	w.Header().Set("Content-Type", "application/json")
	w.Write(val)
}

func getSingleDBOrderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderid := vars["id"]
	order := getDatabaseOrderWithoutStatus(config, orderid)
	val, _ := json.Marshal(order)
	w.Header().Set("Content-Type", "application/json")
	w.Write(val)
}
