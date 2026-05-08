# Stockvacancy API

Backend API untuk sinkronisasi dan publikasi data lowongan kerja dari source API publik ke API internal berbasis Go + MySQL.

## Stack
- Go 1.22
- MySQL 8.4
- Docker + Docker Compose

## Arsitektur
- `cmd/api`: entrypoint aplikasi
- `internal/config`: load environment config
- `internal/database`: koneksi MySQL
- `internal/entity`: entity domain
- `internal/dto`: request/response DTO
- `internal/repository`: interface repository
- `internal/repository/mysql`: implementasi repository MySQL
- `internal/repository/source`: adapter source API publik
- `internal/usecase`: business logic
- `internal/handler`: HTTP handler
- `migrations`: inisialisasi schema MySQL

## Endpoint
- `GET /health`
- `POST /api/v1/sync/jobs`
- `GET /api/v1/jobs?page=1&limit=10&search=backend&location=jakarta&remote=true`
- `GET /api/v1/jobs/{id}`

## Menjalankan aplikasi
```bash
docker compose up --build -d
```

Catatan: service MySQL sengaja tidak diexpose ke host agar tidak bentrok dengan MySQL lokal yang mungkin sudah memakai port 3306.

## Sinkronisasi data job
```bash
curl -X POST http://localhost:8080/api/v1/sync/jobs
```

## Ambil daftar job
```bash
curl "http://localhost:8080/api/v1/jobs?page=1&limit=10"
```

## Ambil detail job
```bash
curl http://localhost:8080/api/v1/jobs/1
```

## Catatan source data
Sekarang source default memakai halaman lowongan Indonesia dari `Kalibrr Indonesia` dan data diambil dari payload `__NEXT_DATA__` halaman publiknya. Untuk memperbanyak hasil, sinkronisasi default memakai multi-keyword:
- software
- backend
- frontend
- mobile
- data
- devops
- golang
- java
- python
- qa

Konfigurasi keyword bisa diubah lewat `SOURCE_QUERIES` pada file `.env` dengan format CSV.

Adapter source dipisahkan agar nanti mudah diganti lagi bila ada source Indonesia lain yang lebih stabil atau lebih resmi.
