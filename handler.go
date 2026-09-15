package main

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database tidak terhubung"})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username dan password wajib diisi"})
		return
	}

	var user User
	err := DB.QueryRow(`
		SELECT id, username, password_hash, role, created_at
		FROM users WHERE username = ?
	`, req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username atau password salah"})
		return
	}

	token, err := GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

func GetVehiclesHandler(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, dealer, brand, model, price, created_at FROM vehicles ORDER BY id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vehicles"})
		return
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		var v Vehicle
		if err := rows.Scan(&v.ID, &v.Dealer, &v.Brand, &v.Model, &v.Price, &v.CreatedAt); err != nil {
			continue
		}
		vehicles = append(vehicles, v)
	}

	c.JSON(http.StatusOK, vehicles)
}

func CreateApplicationHandler(c *gin.Context) {
	userID := c.GetInt64("userID")

	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var vehiclePrice float64
	err := DB.QueryRow(`SELECT price FROM vehicles WHERE id = ?`, req.VehicleID).Scan(&vehiclePrice)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kendaraan tidak ditemukan"})
		return
	}

	if req.DownPayment >= vehiclePrice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uang muka tidak boleh melebihi atau sama dengan harga kendaraan"})
		return
	}

	if req.Tenor <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenor harus lebih dari 0 bulan"})
		return
	}

	installment := req.Installment
	if installment <= 0 {
		principal := vehiclePrice - req.DownPayment
		interest := principal * 0.08 * (float64(req.Tenor) / 12.0)
		installment = math.Round((principal + interest) / float64(req.Tenor))
	}

	var customerID int64
	err = DB.QueryRow(`
		INSERT INTO customers (name, nik, phone, address)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (nik) DO UPDATE SET
			name = excluded.name,
			phone = excluded.phone,
			address = excluded.address
		RETURNING id
	`, req.CustomerName, req.CustomerNIK, req.CustomerPhone, req.CustomerAddress).Scan(&customerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan data debitur: " + err.Error()})
		return
	}

	var appID int64
	err = DB.QueryRow(`
		INSERT INTO applications (customer_id, vehicle_id, created_by, down_payment, tenor, installment, status)
		VALUES (?, ?, ?, ?, ?, ?, 'DRAFT')
		RETURNING id
	`, customerID, req.VehicleID, userID, req.DownPayment, req.Tenor, installment).Scan(&appID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat pengajuan: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          appID,
		"customer_id": customerID,
		"status":      StatusDraft,
		"installment": installment,
		"message":     "Pengajuan berhasil dibuat sebagai DRAFT",
	})
}

func GetApplicationsHandler(c *gin.Context) {
	query := `
		SELECT a.id, a.customer_id, a.vehicle_id, a.created_by, a.down_payment, a.tenor, a.installment,
		       a.status, a.created_at, a.updated_at,
		       c.name, c.nik, c.phone, c.address,
		       v.dealer, v.brand, v.model, v.price
		FROM applications a
		JOIN customers c ON a.customer_id = c.id
		JOIN vehicles v ON a.vehicle_id = v.id
		ORDER BY a.id DESC
	`

	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil daftar pengajuan"})
		return
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var a Application
		var cust Customer
		var veh Vehicle

		err := rows.Scan(
			&a.ID, &a.CustomerID, &a.VehicleID, &a.CreatedBy, &a.DownPayment, &a.Tenor, &a.Installment,
			&a.Status, &a.CreatedAt, &a.UpdatedAt,
			&cust.Name, &cust.NIK, &cust.Phone, &cust.Address,
			&veh.Dealer, &veh.Brand, &veh.Model, &veh.Price,
		)
		if err != nil {
			continue
		}

		cust.ID = a.CustomerID
		veh.ID = a.VehicleID
		a.Customer = &cust
		a.Vehicle = &veh

		apps = append(apps, a)
	}

	c.JSON(http.StatusOK, apps)
}

func GetApplicationByIDHandler(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pengajuan tidak valid"})
		return
	}

	var a Application
	var cust Customer
	var veh Vehicle

	query := `
		SELECT a.id, a.customer_id, a.vehicle_id, a.created_by, a.down_payment, a.tenor, a.installment,
		       a.status, a.created_at, a.updated_at,
		       c.name, c.nik, c.phone, c.address,
		       v.dealer, v.brand, v.model, v.price
		FROM applications a
		JOIN customers c ON a.customer_id = c.id
		JOIN vehicles v ON a.vehicle_id = v.id
		WHERE a.id = ?
	`

	err = DB.QueryRow(query, appID).Scan(
		&a.ID, &a.CustomerID, &a.VehicleID, &a.CreatedBy, &a.DownPayment, &a.Tenor, &a.Installment,
		&a.Status, &a.CreatedAt, &a.UpdatedAt,
		&cust.Name, &cust.NIK, &cust.Phone, &cust.Address,
		&veh.Dealer, &veh.Brand, &veh.Model, &veh.Price,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "pengajuan tidak ditemukan"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mencari pengajuan: " + err.Error()})
		return
	}

	cust.ID = a.CustomerID
	veh.ID = a.VehicleID
	a.Customer = &cust
	a.Vehicle = &veh

	docRows, err := DB.Query(`
		SELECT id, application_id, document_type, file_path, created_at
		FROM application_documents WHERE application_id = ? ORDER BY id ASC
	`, appID)
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var doc ApplicationDocument
			if err := docRows.Scan(&doc.ID, &doc.ApplicationID, &doc.DocumentType, &doc.FilePath, &doc.CreatedAt); err == nil {
				a.Documents = append(a.Documents, doc)
			}
		}
	}

	apprRows, err := DB.Query(`
		SELECT ap.id, ap.application_id, ap.approver_id, u.username, ap.status, ap.notes, ap.approved_at
		FROM application_approvals ap
		JOIN users u ON ap.approver_id = u.id
		WHERE ap.application_id = ? ORDER BY ap.id DESC
	`, appID)
	if err == nil {
		defer apprRows.Close()
		for apprRows.Next() {
			var appr ApplicationApproval
			if err := apprRows.Scan(&appr.ID, &appr.ApplicationID, &appr.ApproverID, &appr.ApproverName, &appr.Status, &appr.Notes, &appr.ApprovedAt); err == nil {
				a.Approvals = append(a.Approvals, appr)
			}
		}
	}

	c.JSON(http.StatusOK, a)
}

func SubmitApplicationHandler(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pengajuan tidak valid"})
		return
	}

	var currentStatus AppStatus
	err = DB.QueryRow(`SELECT status FROM applications WHERE id = ?`, appID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "pengajuan tidak ditemukan"})
		return
	}

	if err := ValidateStatusTransition(currentStatus, StatusSubmitted); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("tidak dapat submit pengajuan dengan status '%s'", currentStatus)})
		return
	}

	_, err = DB.Exec(`
		UPDATE applications SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, StatusSubmitted, appID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal submit pengajuan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      appID,
		"status":  StatusSubmitted,
		"message": "Pengajuan berhasil disubmit ke Supervisor untuk di-review",
	})
}

func ApproveApplicationHandler(c *gin.Context) {
	processApproval(c, StatusApproved)
}

func RejectApplicationHandler(c *gin.Context) {
	processApproval(c, StatusRejected)
}

func processApproval(c *gin.Context, targetStatus AppStatus) {
	idStr := c.Param("id")
	appID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pengajuan tidak valid"})
		return
	}

	approverID := c.GetInt64("userID")

	var req ApprovalRequest
	_ = c.ShouldBindJSON(&req)

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
		return
	}
	defer tx.Rollback()

	var currentStatus AppStatus
	err = tx.QueryRow(`SELECT status FROM applications WHERE id = ?`, appID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "pengajuan tidak ditemukan"})
		return
	}

	if err := ValidateStatusTransition(currentStatus, targetStatus); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("tidak dapat %s pengajuan dengan status '%s'", targetStatus, currentStatus)})
		return
	}

	_, err = tx.Exec(`
		UPDATE applications SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, targetStatus, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengubah status pengajuan"})
		return
	}

	_, err = tx.Exec(`
		INSERT INTO application_approvals (application_id, approver_id, status, notes)
		VALUES (?, ?, ?, ?)
	`, appID, approverID, targetStatus, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mencatat riwayat approval"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal commit transaksi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      appID,
		"status":  targetStatus,
		"message": fmt.Sprintf("Pengajuan berhasil di-%s", targetStatus),
	})
}

func UploadDocumentHandler(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pengajuan tidak valid"})
		return
	}

	var count int
	err = DB.QueryRow(`SELECT COUNT(1) FROM applications WHERE id = ?`, appID).Scan(&count)
	if err != nil || count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Pengajuan #%d belum ada. Silakan buat pengajuan terlebih dahulu.", appID)})
		return
	}

	docType := c.PostForm("document_type")
	if docType == "" {
		docType = "KTP"
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file wajib dipilih"})
		return
	}

	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat folder upload"})
		return
	}

	fileName := fmt.Sprintf("app_%d_%s_%d%s", appID, docType, time.Now().Unix(), filepath.Ext(file.Filename))
	savePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan file"})
		return
	}

	var docID int64
	err = DB.QueryRow(`
		INSERT INTO application_documents (application_id, document_type, file_path)
		VALUES (?, ?, ?)
		RETURNING id
	`, appID, docType, savePath).Scan(&docID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mencatat dokumen: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             docID,
		"application_id": appID,
		"document_type":  docType,
		"file_path":      savePath,
		"message":        "Dokumen berhasil diupload",
	})
}
