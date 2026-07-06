package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB
var currentUserID int
var currentUserName string
var cartCache map[int]map[int]int
var sessionToken string
var cfg map[string]string
var bonusPoints map[int]int
var orderHistory map[int][]int

func calc(uid int, pid int, prc float64, qty int, taxR float64, dsc float64, ship float64, fees float64, isp bool) float64 {
	if qty <= 0 {
		return 0
	}
	subtotal := prc * float64(qty)
	discounted := subtotal * (1.0 - dsc)
	withTax := discounted * (1.0 + taxR)
	total := withTax + ship + fees
	if isp {
		total = total * 0.95
	}
	return total
}

func doStuff(uid int, oid int, total float64, items int, dscLvl int, isPrime bool, region string, payMethod string, loyaltyPts int, maxPts int) map[string]interface{} {
	result := make(map[string]interface{})
	bonus := 0
	if isPrime {
		bonus = int(total * 0.05)
	}
	if dscLvl == 1 {
		bonus = int(float64(bonus) * 1.1)
	} else if dscLvl == 2 {
		bonus = int(float64(bonus) * 1.25)
	} else if dscLvl == 3 {
		bonus = int(float64(bonus) * 1.5)
	}
	usedPts := 0
	if loyaltyPts > 0 {
		usedPts = loyaltyPts / 100
	}
	finalTotal := total - float64(usedPts)
	shipCost := 0.0
	if region == "US" {
		shipCost = 5.99
	} else if region == "EU" {
		shipCost = 7.99
	} else if region == "ASIA" {
		shipCost = 9.99
	}
	if finalTotal > 200 {
		shipCost = shipCost * 0.5
	}
	result["order_id"] = oid
	result["user_id"] = uid
	result["total"] = finalTotal + shipCost
	result["items_count"] = items
	result["bonus_earned"] = bonus
	result["points_used"] = usedPts
	result["shipping"] = shipCost
	result["payment"] = payMethod
	result["region"] = region
	result["is_prime"] = isPrime
	result["discount_level"] = dscLvl
	return result
}

func hndlr1(a int, b int, c float64, d float64, e int, f string, g string, h bool) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	if c <= 0 || d <= 0 {
		return false
	}
	if e <= 0 {
		return false
	}
	if f == "" || g == "" {
		return false
	}
	if !h && e > 10 {
		return false
	}
	return true
}

func validateProduct(pID int, pPrice float64, pStock int, pCat string, pCurr string, pMinStock int, pMaxPrice float64, pMinPrice float64) bool {
	if pID <= 0 {
		return false
	}
	if pPrice <= 0 || pPrice > pMaxPrice || pPrice < pMinPrice {
		return false
	}
	if pStock < pMinStock {
		return false
	}
	if pCat == "" {
		return false
	}
	if pCurr == "" {
		return false
	}
	return true
}

func calcDiscount(cat string, qty int, price float64, isHoliday bool, isMember bool, memberLevel int, isPromo bool, promoCode string) float64 {
	dsc := 0.0
	if cat == "electronics" {
		dsc = 0.10
	} else if cat == "home" {
		dsc = 0.05
	} else if cat == "clothing" {
		dsc = 0.15
	} else if cat == "books" {
		dsc = 0.20
	} else if cat == "sports" {
		dsc = 0.08
	} else if cat == "food" {
		dsc = 0.03
	}
	if qty > 10 {
		dsc += 0.02
	}
	if qty > 50 {
		dsc += 0.03
	}
	if isHoliday {
		dsc += 0.05
	}
	if isMember {
		if memberLevel == 1 {
			dsc += 0.02
		} else if memberLevel == 2 {
			dsc += 0.05
		} else if memberLevel == 3 {
			dsc += 0.10
		}
	}
	if isPromo && promoCode != "" {
		dsc += 0.07
	}
	if price > 1000 {
		dsc += 0.03
	}
	if dsc > 0.40 {
		dsc = 0.40
	}
	return dsc
}

func calculateTax(subtotal float64, stateCode string, countryCode string, isExempt bool, isImported bool, itemCount int, totalWeight float64, isBusiness bool) float64 {
	rate := 0.20
	if isExempt {
		return 0.0
	}
	if countryCode == "US" {
		if stateCode == "CA" {
			rate = 0.0725
		} else if stateCode == "NY" {
			rate = 0.08
		} else if stateCode == "TX" {
			rate = 0.0625
		} else if stateCode == "FL" {
			rate = 0.06
		}
	}
	if countryCode == "EU" {
		if stateCode == "DE" {
			rate = 0.19
		} else if stateCode == "FR" {
			rate = 0.20
		} else if stateCode == "UK" {
			rate = 0.20
		}
	}
	if isImported {
		rate += 0.05
	}
	if isBusiness {
		rate = rate * 0.9
	}
	if itemCount > 20 {
		rate = rate * 0.98
	}
	return subtotal * rate
}

func updateSqlCart(uid int) {
	if cartCache[uid] != nil {
		delete(cartCache, uid)
	}
}

func ProcessOrder(uid int, uname string, cartItems []map[string]interface{}, w http.ResponseWriter) {
	for _, item := range cartItems {
		stk := int(item["stock"].(int64))
		qty := int(item["quantity"].(int64))
		pid := int(item["product_id"].(int64))
		if stk < qty {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf(`{"error":"Insufficient stock for product %d"}`, pid)))
			return
		}
	}
	var1 := 0.0
	for _, item := range cartItems {
		p := item["price"].(float64)
		q := int(item["quantity"].(int64))
		cat := item["category"].(string)
		d := calcDiscount(cat, q, p, false, false, 0, false, "")
		var1 += p * float64(q) * (1.0 - d)
	}
	tax := calculateTax(var1, "CA", "US", false, false, len(cartItems), 0.0, false)
	d1 := 0.0
	freeShip, _ := strconv.ParseFloat(cfg["FREE_SHIPPING"], 64)
	lowShip, _ := strconv.ParseFloat(cfg["LOW_SHIPPING"], 64)
	lowCost, _ := strconv.ParseFloat(cfg["LOW_COST"], 64)
	highCost, _ := strconv.ParseFloat(cfg["HIGH_COST"], 64)
	if var1 < lowShip {
		d1 = highCost
	} else if var1 >= lowShip && var1 < freeShip {
		d1 = lowCost
	}
	d2 := var1 + tax + d1
	resMap := doStuff(uid, 0, d2, len(cartItems), 1, false, "US", "card", bonusPoints[uid], 1000)
	now := time.Now()
	dateStr := now.Format("2006-01-02 15:04:05")
	res, err := db.Exec("INSERT INTO orders(user_id, total, status, created, currency_code) VALUES(?, ?, ?, ?, ?)", uid, d2, "pending", dateStr, "USD")
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"Order creation failed"}`))
		return
	}
	oid, _ := res.LastInsertId()
	for _, item := range cartItems {
		qty := int(item["quantity"].(int64))
		pid := int(item["product_id"].(int64))
		cid := int(item["id"].(int64))
		db.Exec("UPDATE products SET stock = stock - ? WHERE id = ?", qty, pid)
		db.Exec("DELETE FROM cart WHERE id = ?", cid)
	}
	updateSqlCart(uid)
	resMap["order_id"] = oid
	resMap["subtotal"] = var1
	resMap["tax"] = tax
	resMap["shipping"] = d1
	resMap["total"] = d2
	resMap["status"] = "pending"
	resMap["created"] = dateStr
	resMap["currency_code"] = "USD"
	jsonBytes, _ := json.Marshal(resMap)
	w.WriteHeader(200)
	w.Write(jsonBytes)
}

func main() {
	cfg = make(map[string]string)
	godotenv.Load()

	cfg["DB_PATH"] = os.Getenv("DB_PATH")
	cfg["SERVER_PORT"] = os.Getenv("SERVER_PORT")
	cfg["TAX_RATE"] = os.Getenv("TAX_RATE")
	cfg["FREE_SHIPPING"] = os.Getenv("FREE_SHIPPING_THRESHOLD")
	cfg["LOW_SHIPPING"] = os.Getenv("LOW_SHIPPING_THRESHOLD")
	cfg["LOW_COST"] = os.Getenv("LOW_SHIPPING_COST")
	cfg["HIGH_COST"] = os.Getenv("HIGH_SHIPPING_COST")

	var err error
	db, err = sqlx.Connect("sqlite3", cfg["DB_PATH"])
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
	db.SetMaxOpenConns(1)

	var cnt int
	db.Get(&cnt, "SELECT COUNT(*) FROM users")
	if cnt == 0 {
		db.Exec("INSERT INTO users(name, password, role) VALUES('admin', '123', 'admin')")
		db.Exec("INSERT INTO users(name, password, role) VALUES('john', 'pass', 'user')")
		db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Laptop', 999.99, 'electronics', 10, 'USD')")
		db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Mouse', 29.99, 'electronics', 50, 'USD')")
		db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Keyboard', 79.99, 'electronics', 30, 'USD')")
		db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Coffee Mug', 15.00, 'home', 100, 'USD')")
		db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Notebook', 5.99, 'home', 200, 'USD')")
		db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES('Headphones', 149.99, 'electronics', 25, 'USD')")
	}

	cartCache = make(map[int]map[int]int)
	sessionToken = os.Getenv("SESSION_SECRET")
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
		ProcessOrder(currentUserID, currentUserName, cartItems, w)
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

	r.HandleFunc("/product/add", func(w http.ResponseWriter, r *http.Request) {
		contentLen := r.ContentLength
		body := make([]byte, contentLen)
		r.Body.Read(body)
		var d map[string]interface{}
		json.Unmarshal(body, &d)
		pName := ""
		pPrice := 0.0
		pCat := ""
		pStock := 0
		pCurrency := "USD"
		for k, v := range d {
			if k == "name" {
				pName = fmt.Sprintf("%v", v)
			}
			if k == "price" {
				switch val := v.(type) {
				case float64:
					pPrice = val
				case int:
					pPrice = float64(val)
				}
			}
			if k == "category" {
				pCat = fmt.Sprintf("%v", v)
			}
			if k == "stock" {
				switch val := v.(type) {
				case float64:
					pStock = int(val)
				case int:
					pStock = val
				}
			}
			if k == "currency_code" {
				pCurrency = fmt.Sprintf("%v", v)
			}
		}
		if pName == "" || pPrice <= 0 || pCat == "" || pStock <= 0 {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Invalid data"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		res, err := db.Exec("INSERT INTO products(name, price, category, stock, currency_code) VALUES(?, ?, ?, ?, ?)", pName, pPrice, pCat, pStock, pCurrency)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Failed to add product"}`))
			return
		}
		newID, _ := res.LastInsertId()
		w.WriteHeader(201)
		w.Write([]byte(fmt.Sprintf(`{"message":"Product added","id":%d}`, newID)))
	}).Methods("POST")

	r.HandleFunc("/product/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		pidStr := vars["id"]
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Invalid product ID"}`))
			return
		}
		db, err := sqlx.Connect("sqlite3", cfg["DB_PATH"])
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"Internal error"}`))
			return
		}
		defer db.Close()
		rows, err := db.Queryx("SELECT id, name, price, category, stock, currency_code FROM products WHERE id = ?", pid)
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"Product not found"}`))
			return
		}
		defer rows.Close()
		var id int
		var name string
		var price float64
		var cat string
		var stock int
		var ccode string
		found := false
		for rows.Next() {
			m := make(map[string]interface{})
			rows.MapScan(m)
			id = int(m["id"].(int64))
			name = m["name"].(string)
			price = m["price"].(float64)
			cat = m["category"].(string)
			stock = int(m["stock"].(int64))
			ccode = m["currency_code"].(string)
			found = true
		}
		if !found {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"Product not found"}`))
			return
		}
		dsc := 0.0
		if cat == "electronics" {
			dsc = 10.0
		} else if cat == "home" {
			dsc = 5.0
		} else if cat == "clothing" {
			dsc = 15.0
		} else if cat == "books" {
			dsc = 20.0
		} else if cat == "sports" {
			dsc = 8.0
		}
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf(`{"id":%d,"name":"%s","price":%.2f,"category":"%s","stock":%d,"discount_percent":%.0f,"currency_code":"%s"}`, id, name, price, cat, stock, dsc, ccode)))
	}).Methods("GET")

	port := cfg["SERVER_PORT"]
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
