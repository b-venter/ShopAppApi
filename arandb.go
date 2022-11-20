package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	driver "github.com/arangodb/go-driver"
	ahttp "github.com/arangodb/go-driver/http" //Because this http package conflicted with net/http, set statically to ahttp (arango http). aka "import alias"

	"github.com/oklog/ulid/v2" //For generating ULIDs
)

/*ARANGODB*/
//Generic data interface
type d map[string]interface{}

//Item struct
type Item struct {
	Id     string  `json:"id"`
	Name   string  `json:"name"`
	Nett   float32 `json:"nett"`
	Ntt_un string  `json:"nett_unit"`
	Brand  string  `json:"brand"`
}

type ItemNew struct {
	Name   string  `json:"name"`
	Nett   float32 `json:"nett"`
	Ntt_un string  `json:"nett_unit"`
	Brand  string  `json:"brand"`
}

//Shop struct
type Shop struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type ShopNew struct {
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type ShopListsAll struct {
	Id     string  `json:"id"`
	Name   string  `json:"name"`
	Date   float32 `json:"date"`
	Hidden bool    `json:"hidden"`
	Label  string  `json:"label"`
}

type UserNew struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

//The items are returned as an array of arrays containing mixed float and string
//Similar to SlistEdge, but includes edge and vertice (Item) data. Mainly used to map retrieved SList's "items"
type SListItems struct {
	Label     string  `json:"label"`
	Nett      float32 `json:"nett"`
	Nett_unit string  `json:"nett_unit"`
	Price     float32 `json:"price"`
	Currency  string  `json:"currency"`
	Qty       float32 `json:"qty"`
	Special   bool    `json:"special"`
	Trolley   bool    `json:"trolley"`
	Edge_id   string  `json:"edge_id"`
	Item_id   string  `json:"item_id"`
	Shop_id   string  `json:"shop_id"`
}

//ShoppingList struct
type SList struct {
	Shop  string       `json:"shop"`
	Items []SListItems `json:"items"`
}

//ShoppingList Edge: links Shop to Item, along with price, etc
//Mainly used to create new edge doc, or major change to from/to
//{ _to: "Items/382", _from: "Shops/246", date: d, price: 80.20, currency: "NAD", special: false, trolley: false, qty: 6, tag: ""}
type SlistEdge struct {
	To       string  `json:"_to"`
	From     string  `json:"_from"`
	Date     int64   `json:"date"`
	Price    float32 `json:"price"`
	Currency string  `json:"currency"`
	Special  bool    `json:"special"`
	Trolley  bool    `json:"trolley"`
	Qty      float32 `json:"qty"`
	Tag      string  `json:"tag"`
}

//Used for updating edge doc's contents
type SlistEdgeItem struct {
	Date     int64   `json:"date"`
	Price    float32 `json:"price"`
	Currency string  `json:"currency"`
	Special  bool    `json:"special"`
	Trolley  bool    `json:"trolley"`
	Qty      float32 `json:"qty"`
	Tag      string  `json:"tag"`
}

//Price Trend for item
type ItemTrend struct {
	Shop     string `json:"shop"`
	Branch   string `json:"branch"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Currency string `json:"currency"`
	Price    string `json:"price"`
	Date     string `json:"date"`
	List     string `json:"list_id"`
}

//Template items
type TplItem struct {
	Label     string  `json:"label"`
	Nett      float32 `json:"nett"`
	Nett_unit string  `json:"nett_unit"`
	Qty       float32 `json:"qty"`
	Edge_id   string  `json:"edge_id"`
	Item_id   string  `json:"item_id"`
	Shop_id   string  `json:"shop_id"`
	//Tag
}

//Complete template is []Tpl. Array of shops and sub-array of associated items.
type Tpl struct {
	Shop  string    `json:"shop"`
	Items []TplItem `json:"items"`
}

//Similar to SlistEdge, used to create edge document for templates
type TplEdge struct {
	To   string  `json:"_to"`
	From string  `json:"_from"`
	Qty  float32 `json:"qty"`
}

type TplEdgeItem struct {
	Qty float32 `json:"qty"`
}

//Function to generate ULID of new user's db
func makeID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)

	//Ensure ULID always starts with a letter and not a number
	output := "db" + ulid.MustNew(ulid.Timestamp(t), entropy).String()

	return output
}

/*
 * ARANGO DATABASE CONNECTION
 */

func aranDB(x, db string) (driver.Database, context.Context) {

	if !ahok {
		fmt.Println("ArangoDB: Arango DB host not set!")
	}

	conn, err := ahttp.NewConnection(ahttp.ConnectionConfig{
		Endpoints: []string{x},
	})
	if err != nil {
		fmt.Println("ArangoDB: Error creating connection:", err)
	}

	//Create a Client to ArrangoDB
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication("root", ap),
	})
	if err != nil {
		fmt.Println(err)
	}

	if ahok {
		ctx := context.Background()
		db, err := c.Database(ctx, db)
		if err != nil {
			fmt.Println("ArangoDB: Error opening database:", err)
			ct = false
		} else {
			ct = true
			return db, ctx
		}
	}

	return nil, nil
}

/*
* ARANGO CREATE EDGE COLLECTION
 */
func (db dbase) edgeCreate(s string) (driver.Collection, error) {
	//Create Edge collection
	dbx, ctx := aranDB(ah, db.db)

	//Type = 3 for edge, type = 2 for document
	t := &driver.CreateCollectionOptions{Type: 3}
	col, err := dbx.CreateCollection(ctx, s, t)

	if err != nil {
		return nil, err
	}

	return col, nil
}

func dbCreate(x, n string) (string, error) {
	if !ahok {
		fmt.Println("ArangoDB: Arango DB host not set!")
	}

	if n == "" {
		return "", fmt.Errorf("database name cannot be empty")
	}

	conn, err := ahttp.NewConnection(ahttp.ConnectionConfig{
		Endpoints: []string{x},
	})
	if err != nil {
		fmt.Println("ArangoDB: Error creating connection:", err)
	}

	//Create a Client to ArrangoDB
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication("root", ap),
	})
	if err != nil {
		fmt.Println(err)
	}

	if ahok {
		ctx := context.Background()
		op := &driver.CreateDatabaseOptions{}
		dbn, err := c.CreateDatabase(ctx, n, op)
		if err != nil {
			fmt.Println("ArangoDB: Error opening database:", err)
			ct = false
		} else {
			ct = true
			return dbn.Name(), nil
		}
	}

	return "", fmt.Errorf("an unknown error occurred while creatng db")
}

func (db dbase) colCreate(s string) (driver.Collection, error) {
	//Create Edge collection
	dbx, ctx := aranDB(ah, db.db)

	//Type = 3 for edge, type = 2 for document
	t := &driver.CreateCollectionOptions{Type: 2}
	col, err := dbx.CreateCollection(ctx, s, t)

	if err != nil {
		return nil, err
	}

	return col, nil
}

/*
 * ARANGO QUERY METHOD
 */

type aranQuery struct {
	q    string
	bind d
	db   driver.Database
	ctx  context.Context
}

func (query aranQuery) aranQ() []d {
	var ra []d

	cursor, err := query.db.Query(query.ctx, query.q, query.bind)
	if err != nil {
		fmt.Println("ArangoDB Query: Error running query:", err)
	} else {
		for {

			var report d
			_, err := cursor.ReadDocument(query.ctx, &report)

			if driver.IsNoMoreDocuments(err) {
				break
			} else if err != nil {
				fmt.Println(err)
			}

			ra = append(ra, report)
		}
	}

	defer cursor.Close()

	return ra
}

/*
 * ARANGO INSERT METHOD
 */

type aranInsertItem struct {
	cl  string //Specifiy collection name
	in  ItemNew
	db  driver.Database
	ctx context.Context
}

type aranInsertShop struct {
	cl  string //Specifiy collection name
	in  ShopNew
	db  driver.Database
	ctx context.Context
}

func (insert aranInsertItem) aranIns() string {

	//Select collection
	col, err := insert.db.Collection(insert.ctx, insert.cl)
	if err != nil {
		fmt.Println("ArangoDB Insert: Error getting collection", err)
	}

	//Create document
	meta, err := col.CreateDocument(insert.ctx, insert.in)
	if err != nil {
		fmt.Println("ArangoDB Insert: Error creating document", err)
	}

	return meta.Key
}

func (insert aranInsertShop) aranIns() string {

	//Select collection
	col, err := insert.db.Collection(insert.ctx, insert.cl)
	if err != nil {
		fmt.Println("ArangoDB Insert: Error getting collection", err)
	}

	//Create document
	meta, err := col.CreateDocument(insert.ctx, insert.in)
	if err != nil {
		fmt.Println("ArangoDB Insert: Error creating document", err)
	}

	return meta.Key
}

/*
 * ARANGO UPDATE METHOD
 * First get document key. Then update.
 */

type aranUpdateItem struct {
	cl   string //Specifiy collection name
	ky   string //Document key
	data ItemNew
	db   driver.Database
	ctx  context.Context
}

type aranUpdateShop struct {
	cl   string //Specifiy collection name
	ky   string //Document key
	data ShopNew
	db   driver.Database
	ctx  context.Context
}

type aranUpdateSlist struct {
	cl   string //Specifiy collection name
	ky   string //Document key
	data SlistEdgeItem
	db   driver.Database
	ctx  context.Context
}

type aranUpdateSlistAll struct {
	cl   string //Specifiy collection name
	ky   string //Document key
	data ShopListsAll
	db   driver.Database
	ctx  context.Context
}

type aranUpdateTpl struct {
	cl   string //Specifiy collection name
	ky   string //Document key
	data TplEdgeItem
	db   driver.Database
	ctx  context.Context
}

func (update aranUpdateItem) aranUp() (string, error) {
	//Update document based on key with new data

	patch := update.data
	col, err := update.db.Collection(update.ctx, update.cl)
	if err != nil {
		fmt.Println("Error: aranUpdate: db.Collection")
		return "", err
	}

	meta, err := col.UpdateDocument(update.ctx, update.ky, patch)
	if err != nil {
		fmt.Println("Error: aranUpdate: UpdateDocument " + update.ky)
		return "", err
	}

	return meta.Key, nil

}

func (update aranUpdateShop) aranUp() (string, error) {
	//Update document based on key with new data

	patch := update.data
	col, err := update.db.Collection(update.ctx, update.cl)
	if err != nil {
		fmt.Println("Error: aranUpdate: db.Collection")
		return "", err
	}

	meta, err := col.UpdateDocument(update.ctx, update.ky, patch)
	if err != nil {
		fmt.Println("Error: aranUpdate: UpdateDocument " + update.ky)
		return "", err
	}

	return meta.Key, nil

}

func (update aranUpdateSlist) aranUp() (string, error) {
	//Update document based on key with new data

	patch := update.data
	col, err := update.db.Collection(update.ctx, update.cl)
	if err != nil {
		fmt.Println("Error: aranUpdate: db.Collection", err)
		return "", err
	}

	meta, err := col.UpdateDocument(update.ctx, update.ky, patch)
	if err != nil {
		fmt.Println("Error: aranUpdate: UpdateDocument "+update.ky, err, patch)
		return "", err
	}

	return meta.Key, nil

}

func (update aranUpdateSlistAll) aranUp() (string, error) {
	//Update document based on key with new data

	patch := update.data
	col, err := update.db.Collection(update.ctx, update.cl)
	if err != nil {
		fmt.Println("Error: aranUpdate: db.Collection")
		return "", err
	}

	meta, err := col.UpdateDocument(update.ctx, update.ky, patch)
	if err != nil {
		fmt.Println("Error: aranUpdate: UpdateDocument " + update.ky)
		return "", err
	}

	return meta.Key, nil

}

func (update aranUpdateTpl) aranUp() (string, error) {
	//Update document based on key with new data

	patch := update.data
	col, err := update.db.Collection(update.ctx, update.cl)
	if err != nil {
		fmt.Println("Error: aranUpdate: db.Collection", err)
		return "", err
	}

	meta, err := col.UpdateDocument(update.ctx, update.ky, patch)
	if err != nil {
		fmt.Println("Error: aranUpdate: UpdateDocument "+update.ky, err, patch)
		return "", err
	}

	return meta.Key, nil

}

/*
 * ARANGO GET DOCUMENT KEY METHOD(S)
 *  Dependent on aranQuery func
 */

//Get document key based on Month Year
type getDocKey struct {
	month int
	year  int
	ulid  string
	db    driver.Database
	ctx   context.Context
}

func (my getDocKey) getKey() []d {
	//Get key WHERE month, year IN ulid
	var query string = "FOR doc IN " + my.ulid + " FILTER doc.`year` == @year AND doc.`month` == @month RETURN { 'key':doc._key }"
	var bind = d{"year": my.year, "month": my.month}
	var key []d

	keys := aranQuery{query, bind, my.db, my.ctx}
	key = keys.aranQ()

	return key
}
