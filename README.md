# BCA Finance — Vehicle Credit Application & Approval MVP (SQLite)

Sistem berbasis Web dan REST API untuk proses digitalisasi pengajuan kredit kendaraan bermotor dan approval pengajuan (Pilihan Soal 2B).

---

## 1. Database: SQLite (`bca_finance.sqlite`)

Aplikasi menggunakan embedded database SQLite murni (tanpa setup database terpisah). File database `bca_finance.sqlite` otomatis dibuat saat server dijalankan.

---

## 2. Cara Menjalankan

### Langkah 1: Unduh Dependency

```bash
go mod tidy
```

### Langkah 2: Jalankan Server

```bash
go run .
```

Server akan aktif di: <http://localhost:8080>

---

## 3. Akun Default untuk Login

* **Sales:** Username: `sales1` | Password: `password123`
* **Approver:** Username: `approver1` | Password: `password123`

---

## 4. Alur Bisnis yang Diuji

1. Login sebagai `sales1`
2. Isi form pengajuan kredit baru (nama, NIK, kendaraan, DP, tenor) -> tersimpan sebagai `DRAFT`
3. Upload berkas KTP / KK / SPK
4. Klik **Submit ke Supervisor** -> status berubah menjadi `SUBMITTED`
5. Login sebagai `approver1` (atau buka halaman Approval)
6. Klik **Review & Putuskan** pada pengajuan
7. Tulis catatan, lalu klik **SETUJUI (APPROVE)** atau **TOLAK (REJECT)**
8. Status pengajuan diperbarui dan tercatat dalam riwayat approval

---

## 5. Unit Test State Machine

Untuk memverifikasi aturan transisi status:

```bash
go test -v ./...
```
