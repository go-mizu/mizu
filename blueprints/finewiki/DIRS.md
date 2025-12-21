finewiki/
├── cmd/
│   └── finewiki/
│       └── main.go
│
├── app/
│   └── web/
│       ├── server.go
│       ├── handlers.go
│       ├── render.go
│       └── middleware.go
│
├── feature/
│   ├── search/
│   │   ├── api.go
│   │   └── service.go
│   │
│   └── view/
│       ├── api.go
│       └── service.go
│
├── store/
│   └── duckdb/
│       ├── schema.sql
│       ├── seed.sql
│       ├── store.go
│       └── import.go
│
├── views/
│   ├── layout/
│   │   └── app.html
│   │
│   ├── component/
│   │   ├── topbar.html
│   │   ├── sidebar.html
│   │   └── chips.html
│   │
│   └── page/
│       ├── home.html
│       ├── search.html
│       └── view.html
│
├── go.mod
└── README.md
