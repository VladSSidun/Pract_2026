package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var testDB *sqlx.DB

func setupTestDB(t *testing.T) {
	os.Remove("./test_store.db")
	var err error
	testDB, err = sqlx.Connect("sqlite3", "./test_store.db")
	if err != nil {
		t.Fatal(err)
	}
	testDB.Exec(`CREATE TABLE IF NOT EXISTS users(id INTEGER PRIMARY KEY, name TEXT, password TEXT, role TEXT)`)
	testDB.Exec(`CREATE TABLE IF NOT EXISTS products(id INTEGER PRIMARY KEY, name TEXT, price REAL, category TEXT, stock INTEGER, currency_code TEXT)`)
	testDB.Exec(`CREATE TABLE IF NOT EXISTS cart(id INTEGER PRIMARY KEY, user_id INTEGER, product_id INTEGER, quantity INTEGER)`)
	testDB.Exec(`CREATE TABLE IF NOT EXISTS orders(id INTEGER PRIMARY KEY, user_id INTEGER, total REAL, status TEXT, created TEXT, currency_code TEXT)`)
	testDB.Exec("INSERT INTO users(name, password, role) VALUES('testuser', 'testpass', 'user')")
	testDB.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Test Product', 99.99, 'electronics', 100, 'USD')")
}

func setupTestRouter() *mux.Router {
	cfg = make(map[string]string)
	cfg["DB_PATH"] = "./test_store.db"
	cfg["SERVER_PORT"] = "8080"
	cfg["TAX_RATE"] = "0.20"
	cfg["FREE_SHIPPING"] = "150.00"
	cfg["LOW_SHIPPING"] = "50.00"
	cfg["LOW_COST"] = "4.99"
	cfg["HIGH_COST"] = "9.99"

	currentUserID = 0
	currentUserName = ""
	cartCache = make(map[int]map[int]int)
	bonusPoints = make(map[int]int)
	orderHistory = make(map[int][]int)

	r := mux.NewRouter()

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			w.Write([]byte(`{"error":"Method not allowed"}`))
			return
		}
		contentLen := r.ContentLength
		body := make([]byte, contentLen)
		r.Body.Read(body)
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)
		username := ""
		password := ""
		for k, v := range parsed {
			if k == "username" {
				username = fmt.Sprintf("%v", v)
			}
			if k == "password" {
				password = fmt.Sprintf("%v", v)
			}
		}
		if username == "" || password == "" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Missing credentials"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT id, name FROM users WHERE name = ? AND password = ?", username, password)
		if err != nil {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Invalid credentials"}`))
			return
		}
		defer rows.Close()
		found := false
		var uid int
		var uname string
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			uid = int(m["id"].(int64))
			uname = m["name"].(string)
			found = true
		}
		if !found {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Invalid credentials"}`))
			return
		}
		currentUserID = uid
		currentUserName = uname
		sessionToken = fmt.Sprintf("token-%d-%d", uid, time.Now().Unix())
		w.WriteHeader(200)
		jsonData, _ := json.Marshal(map[string]interface{}{"message": "Login successful", "user_id": uid, "user_name": uname, "token": sessionToken})
		w.Write(jsonData)
	}).Methods("POST")

	r.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT id, name, price, category, stock, currency_code FROM products")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Database error"}`))
			return
		}
		defer rows.Close()
		result := "["
		first := true
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			id := m["id"].(int64)
			name := m["name"].(string)
			price := m["price"].(float64)
			cat := m["category"].(string)
			stock := m["stock"].(int64)
			ccode := m["currency_code"].(string)
			item := fmt.Sprintf(`{"id":%d,"name":"%s","price":%.2f,"category":"%s","stock":%d,"currency_code":"%s"}`, id, name, price, cat, stock, ccode)
			if first {
				result += item
				first = false
			} else {
				result += "," + item
			}
		}
		result += "]"
		w.WriteHeader(200)
		w.Write([]byte(result))
	}).Methods("GET")

	r.HandleFunc("/cart/add", func(w http.ResponseWriter, r *http.Request) {
		contentLen := r.ContentLength
		body := make([]byte, contentLen)
		r.Body.Read(body)
		var data map[string]interface{}
		json.Unmarshal(body, &data)
		prodID := 0
		qty := 1
		for k, v := range data {
			if k == "product_id" {
				switch val := v.(type) {
				case float64:
					prodID = int(val)
				case int:
					prodID = val
				}
			}
			if k == "quantity" {
				switch val := v.(type) {
				case float64:
					qty = int(val)
				case int:
					qty = val
				}
			}
		}
		if prodID == 0 {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Invalid product_id"}`))
			return
		}
		if currentUserID == 0 {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Not logged in"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT id, name, price, category, stock, currency_code FROM products WHERE id = ?", prodID)
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"Product not found"}`))
			return
		}
		defer rows.Close()
		var pName string
		var pPrice float64
		var pCat string
		var pStock int
		var pCurrency string
		found := false
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			pName = m["name"].(string)
			pPrice = m["price"].(float64)
			pCat = m["category"].(string)
			pStock = int(m["stock"].(int64))
			pCurrency = m["currency_code"].(string)
			found = true
		}
		if !found {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"Product not found"}`))
			return
		}
		valid := validateProduct(prodID, pPrice, pStock, pCat, pCurrency, 0, 999999.99, 0.01)
		if !valid {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Invalid product data"}`))
			return
		}
		if pStock < qty {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Not enough stock"}`))
			return
		}
		discount := calcDiscount(pCat, qty, pPrice, false, false, 0, false, "")
		finalPrice := pPrice * float64(qty) * (1.0 - discount)
		taxRateStr := cfg["TAX_RATE"]
		taxRate, _ := strconv.ParseFloat(taxRateStr, 64)
		if strings.HasSuffix(pCat, "food") {
			taxRate = 0.07
		}
		totalWithTax := finalPrice * (1.0 + taxRate)
		existingRows, err := db.Queryx("SELECT id, quantity FROM cart WHERE user_id = ? AND product_id = ?", currentUserID, prodID)
		if err == nil {
			var cartID int
			var existingQty int
			for existingRows.Next() {
				m := make(map[string]interface{})
				existingRows.MapScan(m)
				cartID = int(m["id"].(int64))
				existingQty = int(m["quantity"].(int64))
			}
			existingRows.Close()
			if cartID > 0 {
				newQty := existingQty + qty
				db.Exec("UPDATE cart SET quantity = ? WHERE id = ?", newQty, cartID)
			} else {
				db.Exec("INSERT INTO cart(user_id, product_id, quantity) VALUES(?, ?, ?)", currentUserID, prodID, qty)
			}
		} else {
			db.Exec("INSERT INTO cart(user_id, product_id, quantity) VALUES(?, ?, ?)", currentUserID, prodID, qty)
		}
		if cartCache[currentUserID] == nil {
			cartCache[currentUserID] = make(map[int]int)
		}
		cartCache[currentUserID][prodID] += qty
		w.WriteHeader(200)
		resp := map[string]interface{}{
			"message":       "Added to cart",
			"product_name":  pName,
			"quantity":      qty,
			"unit_price":    pPrice,
			"discount":      discount * 100,
			"final_price":   finalPrice,
			"tax":           taxRate * 100,
			"total":         totalWithTax,
			"currency_code": pCurrency,
		}
		jsonBytes, _ := json.Marshal(resp)
		w.Write(jsonBytes)
	}).Methods("POST")

	r.HandleFunc("/cart", func(w http.ResponseWriter, r *http.Request) {
		if currentUserID == 0 {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Not logged in"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT c.id, c.product_id, c.quantity, p.name, p.price, p.category, p.currency_code FROM cart c JOIN products p ON c.product_id = p.id WHERE c.user_id = ?", currentUserID)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Database error"}`))
			return
		}
		defer rows.Close()
		var total float64 = 0.0
		var itemsJSON []string
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			cid := int(m["id"].(int64))
			pid := int(m["product_id"].(int64))
			qty := int(m["quantity"].(int64))
			name := m["name"].(string)
			price := m["price"].(float64)
			cat := m["category"].(string)
			ccode := m["currency_code"].(string)
			dsc := calcDiscount(cat, qty, price, false, false, 0, false, "")
			subtotal := price * float64(qty) * (1.0 - dsc)
			total += subtotal
			itemJSON := fmt.Sprintf(`{"cart_id":%d,"product_id":%d,"name":"%s","quantity":%d,"unit_price":%.2f,"discount_pct":%.0f,"subtotal":%.2f,"currency_code":"%s"}`, cid, pid, name, qty, price, dsc*100, subtotal, ccode)
			itemsJSON = append(itemsJSON, itemJSON)
		}
		taxRateStr := cfg["TAX_RATE"]
		taxRate, _ := strconv.ParseFloat(taxRateStr, 64)
		grandTotal := total * (1.0 + taxRate)
		if total > 500 {
			grandTotal = total*1.18 + (total * 0.02)
		} else if total > 200 {
			grandTotal = total * 1.19
		} else {
			grandTotal = total * (1.0 + taxRate)
		}
		result := fmt.Sprintf(`{"user_id":%d,"user_name":"%s","items":[%s],"subtotal":%.2f,"tax_rate":%.2f,"grand_total":%.2f,"currency_code":"USD"}`, currentUserID, currentUserName, strings.Join(itemsJSON, ","), total, taxRate*100, grandTotal)
		w.WriteHeader(200)
		w.Write([]byte(result))
	}).Methods("GET")

	r.HandleFunc("/cart/checkout", func(w http.ResponseWriter, r *http.Request) {
		if currentUserID == 0 {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Not logged in"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT c.id, c.product_id, c.quantity, p.price, p.stock, p.category, p.currency_code FROM cart c JOIN products p ON c.product_id = p.id WHERE c.user_id = ?", currentUserID)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Database error"}`))
			return
		}
		defer rows.Close()
		var cartItems []map[string]interface{}
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			cartItems = append(cartItems, m)
		}
		for _, item := range cartItems {
			stock := int(item["stock"].(int64))
			qty := int(item["quantity"].(int64))
			pid := int(item["product_id"].(int64))
			if stock < qty {
				w.WriteHeader(400)
				w.Write([]byte(fmt.Sprintf(`{"error":"Insufficient stock for product %d"}`, pid)))
				return
			}
		}
		var total float64 = 0.0
		for _, item := range cartItems {
			price := item["price"].(float64)
			qty := int(item["quantity"].(int64))
			cat := item["category"].(string)
			dsc := calcDiscount(cat, qty, price, false, false, 0, false, "")
			itemTotal := price * float64(qty) * (1.0 - dsc)
			total += itemTotal
		}
		tax := calculateTax(total, "CA", "US", false, false, len(cartItems), 0.0, false)
		shipping := 0.0
		freeShip, _ := strconv.ParseFloat(cfg["FREE_SHIPPING"], 64)
		lowShip, _ := strconv.ParseFloat(cfg["LOW_SHIPPING"], 64)
		lowCost, _ := strconv.ParseFloat(cfg["LOW_COST"], 64)
		highCost, _ := strconv.ParseFloat(cfg["HIGH_COST"], 64)
		if total < lowShip {
			shipping = highCost
		} else if total >= lowShip && total < freeShip {
			shipping = lowCost
		} else {
			shipping = 0.0
		}
		grandTotal := total + tax + shipping
		now := time.Now()
		dateStr := now.Format("2006-01-02 15:04:05")
		res, err := db.Exec("INSERT INTO orders(user_id, total, status, created, currency_code) VALUES(?, ?, ?, ?, ?)", currentUserID, grandTotal, "pending", dateStr, "USD")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Order creation failed"}`))
			return
		}
		orderID, _ := res.LastInsertId()
		for _, item := range cartItems {
			qty := int(item["quantity"].(int64))
			pid := int(item["product_id"].(int64))
			cid := int(item["id"].(int64))
			db.Exec("UPDATE products SET stock = stock - ? WHERE id = ?", qty, pid)
			db.Exec("DELETE FROM cart WHERE id = ?", cid)
		}
		if cartCache[currentUserID] != nil {
			delete(cartCache, currentUserID)
		}
		w.WriteHeader(200)
		resp := fmt.Sprintf(`{"order_id":%d,"user_id":%d,"subtotal":%.2f,"tax":%.2f,"shipping":%.2f,"total":%.2f,"status":"pending","created":"%s","currency_code":"USD"}`, orderID, currentUserID, total, tax, shipping, grandTotal, dateStr)
		w.Write([]byte(resp))
	}).Methods("POST")

	r.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if currentUserID == 0 {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"Not logged in"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT id, user_id, total, status, created, currency_code FROM orders WHERE user_id = ?", currentUserID)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Database error"}`))
			return
		}
		defer rows.Close()
		result := "["
		first := true
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			oid := m["id"].(int64)
			uid := m["user_id"].(int64)
			total := m["total"].(float64)
			status := m["status"].(string)
			created := m["created"].(string)
			ccode := m["currency_code"].(string)
			item := fmt.Sprintf(`{"order_id":%d,"user_id":%d,"total":%.2f,"status":"%s","created":"%s","currency_code":"%s"}`, oid, uid, total, status, created, ccode)
			if first {
				result += item
				first = false
			} else {
				result += "," + item
			}
		}
		result += "]"
		w.WriteHeader(200)
		w.Write([]byte(result))
	}).Methods("GET")

	return r
}

func TestAllEndpoints(t *testing.T) {
	setupTestDB(t)
	defer os.Remove("./test_store.db")

	router := setupTestRouter()
	defer testDB.Close()

	t.Run("TestLoginSuccess", func(t *testing.T) {
		body := map[string]string{"username": "testuser", "password": "testpass"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Login successful") {
			t.Errorf("Expected 'Login successful' in response")
		}
		if !strings.Contains(rr.Body.String(), "user_id") {
			t.Errorf("Expected 'user_id' in response")
		}
		if !strings.Contains(rr.Body.String(), "token") {
			t.Errorf("Expected 'token' in response")
		}
	})

	t.Run("TestLoginInvalidCredentials", func(t *testing.T) {
		body := map[string]string{"username": "wrong", "password": "wrong"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 401 {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("TestLoginMissingCredentials", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 400 {
			t.Errorf("Expected 400, got %d", rr.Code)
		}
	})

	t.Run("TestProductsList", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/products", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Test Product") {
			t.Errorf("Expected 'Test Product' in response")
		}
		if !strings.Contains(rr.Body.String(), "99.99") {
			t.Errorf("Expected price '99.99' in response")
		}
	})

	t.Run("TestCartAddWithoutLogin", func(t *testing.T) {
		currentUserID = 0
		body := map[string]interface{}{"product_id": 1, "quantity": 2}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/cart/add", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 401 {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("TestCartAddInvalidProduct", func(t *testing.T) {
		currentUserID = 1
		body := map[string]interface{}{"product_id": 999, "quantity": 1}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/cart/add", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 404 {
			t.Errorf("Expected 404, got %d", rr.Code)
		}
	})

	t.Run("TestCartAddSuccess", func(t *testing.T) {
		currentUserID = 1
		body := map[string]interface{}{"product_id": 1, "quantity": 2}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/cart/add", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Added to cart") {
			t.Errorf("Expected 'Added to cart' in response")
		}
		if !strings.Contains(rr.Body.String(), "currency_code") {
			t.Errorf("Expected 'currency_code' in response")
		}
	})

	t.Run("TestCartView", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/cart", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "Test Product") {
			t.Errorf("Expected 'Test Product' in cart")
		}
		if !strings.Contains(rr.Body.String(), "user_id") {
			t.Errorf("Expected 'user_id' in response")
		}
	})

	t.Run("TestCartViewUnauthorized", func(t *testing.T) {
		currentUserID = 0
		req, _ := http.NewRequest("GET", "/cart", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 401 {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("TestCheckoutSuccess", func(t *testing.T) {
		currentUserID = 1
		testDB.Exec("INSERT INTO cart(user_id, product_id, quantity) VALUES(1, 1, 1)")
		req, _ := http.NewRequest("POST", "/cart/checkout", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "order_id") {
			t.Errorf("Expected 'order_id' in response")
		}
		if !strings.Contains(rr.Body.String(), "currency_code") {
			t.Errorf("Expected 'currency_code' in response")
		}
	})

	t.Run("TestCheckoutUnauthorized", func(t *testing.T) {
		currentUserID = 0
		req, _ := http.NewRequest("POST", "/cart/checkout", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 401 {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	t.Run("TestOrdersList", func(t *testing.T) {
		currentUserID = 1
		req, _ := http.NewRequest("GET", "/orders", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("Expected 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "order_id") {
			t.Errorf("Expected 'order_id' in response")
		}
	})

	t.Run("TestWrongMethodLogin", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 405 {
			t.Errorf("Expected 405, got %d", rr.Code)
		}
	})

	t.Run("TestWrongMethodProducts", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/products", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != 405 {
			t.Errorf("Expected 405, got %d", rr.Code)
		}
	})

	t.Run("TestHelperFunctions", func(t *testing.T) {
		result := validateProduct(1, 100.0, 10, "electronics", "USD", 0, 1000.0, 0.01)
		if !result {
			t.Errorf("validateProduct should return true for valid product")
		}

		result = validateProduct(0, 100.0, 10, "electronics", "USD", 0, 1000.0, 0.01)
		if result {
			t.Errorf("validateProduct should return false for invalid id")
		}

		dsc := calcDiscount("electronics", 5, 100.0, false, false, 0, false, "")
		if dsc <= 0 {
			t.Errorf("calcDiscount should return positive value for electronics")
		}
	})

	t.Run("TestCalculateTax", func(t *testing.T) {
		tax := calculateTax(100.0, "CA", "US", false, false, 5, 0.0, false)
		if tax <= 0 {
			t.Errorf("calculateTax should return positive value")
		}

		tax = calculateTax(100.0, "CA", "US", true, false, 5, 0.0, false)
		if tax != 0 {
			t.Errorf("calculateTax should return 0 for exempt items")
		}
	})

	t.Run("TestDeadCodeFunctions", func(t *testing.T) {
		result := calc(1, 1, 100.0, 2, 0.2, 0.1, 5.99, 0.0, false)
		if result <= 0 {
			t.Errorf("calc should return positive value")
		}

		m := process(1, 2, 10.0, 20.0, 5, "a", "b", 3)
		if m["res1"] == nil {
			t.Errorf("process should return map with results")
		}

		ok := hndlr1(1, 2, 10.0, 20.0, 5, "x", "y", true)
		if !ok {
			t.Errorf("hndlr1 should return true for valid args")
		}

		ok = hndlr1(0, 2, 10.0, 20.0, 5, "x", "y", true)
		if ok {
			t.Errorf("hndlr1 should return false for invalid args")
		}
	})
}
