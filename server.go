package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"net/http"
	"os"
	_ "strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

/*ARANGODB SECTION*/
/* Sensitive/custom settings */
//Arangodb Host
var ah, ahok = os.LookupEnv("ADB_HOST")
var ap, _ = os.LookupEnv("ADB_PASS")

//Connection test
var ct bool = false

//Local cache of users, db and jwt sub
type user struct {
	email string
	db    string
	role  string
}

type Cache map[string]user

//Initiate "cache" - if this ever gets BIG, then rather use redis or even just go-cache
//["sub"]{user struct}
var cache = make(Cache)

//Simple db type for methods
type dbase struct {
	db string
}

/* end */

func adminMaybe(c echo.Context) error {
	//Get db from context, convert from interface to string
	cta := fmt.Sprintf("%v", c.Request().Context().Value("sub"))
	r := cache[cta].role

	if r == "admin" {
		return c.JSON(http.StatusOK, "admin")
	} else {
		return c.JSON(http.StatusOK, "user")
	}
}

/* #############
 * GET Functions
 * #############
 */

//Common function to run getQueries
//q is the query
//b1 is the @bind, b2 is the value
func (db dbase) getQueries(q, b1, b2 string) ([]d, error) {

	//Some queries have no bind. In that case, create as blank, or nil. If present, assign bind
	var bind d
	if b1 != "" {
		bind = d{b1: b2}
	}

	dbx, ctx := aranDB(ah, db.db)

	var execQ []d

	if ct {
		data := aranQuery{q, bind, dbx, ctx}
		execQ = data.aranQ()
	} else {
		fmt.Println("Failed to connect. Troubleshoot connection to ", ah)
		var err = errors.New("failed to connect to db")
		return nil, err
	}

	return execQ, nil
}

func (db dbase) getShoppingList(id string) (string, error) {

	//DB query - get ShoppingList name
	query := "FOR a in ShoppingLists FILTER a._key == @id RETURN {'edge': a.name}"

	slistQ, err := db.getQueries(query, "id", id)

	//Catch error from the query
	if err != nil {
		return "", err
	}

	if slistQ == nil {
		return "", errors.New("no such id")
	}

	//Convert received value from interface to string
	sl := fmt.Sprint(slistQ[0]["edge"])

	return sl, nil

}

func (db dbase) getTemplate(id string) (string, error) {

	//DB query - get ShoppingList name
	query := "FOR a in Templates FILTER a._key == @id RETURN {'edge': a.name}"

	slistQ, err := db.getQueries(query, "id", id)

	//Catch error from the query
	if err != nil {
		return "", err
	}

	//Convert received value from interface to string
	sl := fmt.Sprint(slistQ[0]["edge"])

	return sl, nil

}

func itemGetSpecific(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	id = "Items/" + id

	//DB query
	query := "FOR item IN Items FILTER item._id == @itemID RETURN { 'id': item._key, 'name': item.name, 'nett': item.nett, 'nett_unit': item.nett_unit, 'brand': item.brand }"

	//Run query and response
	execQ, err := db.getQueries(query, "itemID", id)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Blank data returned indicates request was for a non-existent id
	if execQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	//All good, send 200OK and data
	return c.JSON(http.StatusOK, execQ[0])

}

func shopGetSpecific(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	id = "Shops/" + id

	//DB query
	query := "FOR shop IN Shops FILTER shop._id == @shopID RETURN { 'id': shop._key, 'name': shop.name, 'branch': shop.branch, 'city': shop.city, 'country': shop.country }"

	//Run query and response
	execQ, err := db.getQueries(query, "shopID", id)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Blank data returned indicates request was for a non-existent id
	if execQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	//All good, send 200OK and data
	return c.JSON(http.StatusOK, execQ[0])

}

func itemGetAll(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//DB query
	query := "FOR item IN Items RETURN { 'id': item._key, 'name': item.name, 'nett': item.nett, 'nett_unit': item.nett_unit, 'brand': item.brand }"
	var bind string

	//Run query and response. bind is a null string "" since no binding takes place for the query
	execQ, err := db.getQueries(query, bind, bind)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if execQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, execQ)

}

func shopGetAll(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//DB query
	query := "FOR shop IN Shops RETURN { 'id': shop._key, 'name': shop.name, 'branch': shop.branch, 'city': shop.city, 'country': shop.country }"
	var bind string

	//Run query and response. bind is a null string "" since no binding takes place for the query
	execQ, err := db.getQueries(query, bind, bind)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if execQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, execQ)
}

//TODO: LIKE() with caseInsensitive. https://www.arangodb.com/docs/3.9/aql/functions-string.html#like
func itemGetLike(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	find := "%" + c.Param("part") + "%"

	//DB query
	query := "FOR item IN Items FILTER item.name LIKE @find RETURN { 'id': item._key, 'name': item.name, 'nett': item.nett, 'nett_unit': item.nett_unit, 'brand': item.brand }"

	//Run query and response
	execQ, err := db.getQueries(query, "find", find)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if execQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, execQ)

}

func shopGetLike(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	find := "%" + c.Param("part") + "%"

	//DB query
	query := "FOR shop IN Shops FILTER shop.name LIKE @find RETURN { 'id': shop._key, 'name': shop.name, 'branch': shop.branch, 'city': shop.city, 'country': shop.country }"

	//Run query and response
	execQ, err := db.getQueries(query, "find", find)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if execQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, execQ)
}

func listGetVisible(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//DB query
	query := "FOR list in ShoppingLists FILTER list.hidden == false RETURN {'name': list.name, 'date': list.date, 'id': list._key, 'label': list.label}"
	var bind string

	//Run query and response. bind is a null string "" since no binding takes place for the query
	listQ, err := db.getQueries(query, bind, bind)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if listQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, listQ)
}

//ShoppingList id - i, database - d,p - query params
func getNameCore(i, d, p string) ([]d, error) {
	db := dbase{d}

	var bind string
	bind2 := i
	var query string

	//DB query
	if p == "sl" {
		query = "FOR sl in ShoppingLists FILTER sl._key == @sl RETURN {'label': sl.label}"
		bind = "sl"
	} else if p == "tpl" {
		query = "FOR tpl in Templates FILTER tpl._key == @tpl RETURN {'label': tpl.label}"
		bind = "tpl"
	}

	//Run query and response.
	listQ, err := db.getQueries(query, bind, bind2)

	return listQ, err

}

func listGetName(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))

	//Get item id
	id := c.Param("id")

	listQ, err := getNameCore(id, dbv, "sl")

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if listQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, listQ)
}

func listTemplateName(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))

	//Get item id
	id := c.Param("id")

	listQ, err := getNameCore(id, dbv, "tpl")

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if listQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, listQ)
}

func listGetAll(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//DB query
	query := "FOR list in ShoppingLists RETURN {'name': list.name, 'date': list.date, 'hidden': list.hidden, 'id': list._key, 'label': list.label}"
	var bind string

	//Run query and response. bind is a null string "" since no binding takes place for the query
	listQ, err := db.getQueries(query, bind, bind)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if listQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, listQ)
}

func listGetShopping(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))

	//Get item id
	id := c.Param("id")

	shQ, err := listGetShoppingCore(id, dbv, "full")

	//Catch errors
	if err != nil {
		fmt.Println("ShoppingList error2:", err)

		if err.Error() == "id error" {
			return c.JSON(http.StatusBadRequest, "invalid id")
		} else if err.Error() == "server error" {
			return c.JSON(http.StatusInternalServerError, "server error")
		}
	}

	if shQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, shQ)
}

//ShoppingList id - i, database - d,p - query params
func listGetShoppingCore(i, dbv, p string) ([]d, error) {
	//Assign database
	db := dbase{dbv}

	//Get ShoppingList id
	id := i

	s, err := db.getShoppingList(id)
	if err != nil {
		return nil, errors.New("id error")
	}

	//DB query - get shopping list contents
	//"p" is query parameter
	// - "full" returns the full ShoppingList Edge Item
	// - "qty" returns only Template Edge item (see listMakeTemplate)
	var query2 string
	var qb string
	if p == "full" {
		query2 = "FOR c in Shops let b = c.name let sub = (FOR v, e IN 1..1 OUTBOUND c @slist let a = {'label': v.name, 'nett': v.nett, 'nett_unit': v.nett_unit, 'price': e.price, 'currency': e.currency, 'qty': e.qty, 'trolley': e.trolley, 'special': e.special, 'edge_id': e._key, 'item_id': v._key, 'shop_id': c._key} RETURN a ) FILTER sub != [] RETURN {'shop': b, 'items': sub}"
		qb = "slist"
	} else if p == "qty" {
		query2 = "FOR c in Shops let b = c.name let sub = (FOR v, e IN 1..1 OUTBOUND c @tpl let a = {'label': v.name, 'nett': v.nett, 'nett_unit': v.nett_unit, 'qty': e.qty, 'edge_id': e._key, 'item_id': v._key, 'shop_id': c._key} RETURN a ) FILTER sub != [] RETURN {'shop': b, 'items': sub}"
		qb = "tpl"
	}

	shQ, err2 := db.getQueries(query2, qb, s)

	//Catch error from the query
	if err2 != nil {
		return nil, errors.New("server error")
	}

	if shQ == nil {
		return nil, nil
	}

	return shQ, nil
}

func listGetTrolley(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")

	//Get Shop id
	sh := c.Param("key")

	s, err := db.getShoppingList(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	shop := "Shops/" + sh

	//DB query - get shopping list contents
	query := "FOR v, e IN 1..1 OUTBOUND '" + shop + "' @slist let a = {'label': v.name, 'nett': v.nett, 'nett_unit': v.nett_unit, 'price': e.price, 'currency': e.currency, 'qty': e.qty, 'trolley': e.trolley, 'special': e.special, 'edge_id': e._key, 'item_id': v._key} FILTER e.trolley == true RETURN a"

	shQ, err := db.getQueries(query, "slist", s)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	if shQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, shQ)
}

func listGetTemplates(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//DB query
	query := "FOR tpl in Templates RETURN {'name': tpl.name, 'date': tpl.date, 'hidden': tpl.hidden, 'id': tpl._key, 'label': tpl.label}"
	var bind string

	//Run query and response. bind is a null string "" since no binding takes place for the query
	listQ, err := db.getQueries(query, bind, bind)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
		//TODO: return specific error for "AQL: collection or view not found: Templates (while parsing)"
		// This will allow frontend code to differentiate between 500 errors
	}

	if listQ == nil {
		fault := "No data returned"
		return c.JSON(http.StatusBadRequest, fault)
	}

	return c.JSON(http.StatusOK, listQ)

}

func listTemplateDetailsCore(i, dbv string) ([]d, error) {

	db := dbase{dbv}
	s, err := db.getTemplate(i)
	if err != nil {
		return nil, errors.New("id error")
	}

	//DB query - get template's contents
	query2 := "FOR c in Shops let b = c.name let sub = (FOR v, e IN 1..1 OUTBOUND c @tpl let a = {'label': v.name, 'nett': v.nett, 'nett_unit': v.nett_unit, 'qty': e.qty, 'edge_id': e._key, 'item_id': v._key, 'shop_id': c._key} RETURN a ) FILTER sub != [] RETURN {'shop': b, 'items': sub}"

	tplQ, err := db.getQueries(query2, "tpl", s)

	//Catch error from the query
	if err != nil {
		return nil, errors.New("server error")
	}

	return tplQ, nil

}

func listTemplateDetails(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))

	//Get item id
	id := c.Param("id")

	tplQ, err := listTemplateDetailsCore(id, dbv)

	//Catch errors
	if err != nil {
		if err.Error() == "id error" {
			return c.JSON(http.StatusBadRequest, "invalid id")
		} else if err.Error() == "server error" {
			return c.JSON(http.StatusInternalServerError, "server error")
		}
	}

	if tplQ == nil {
		fault := "No data to return."
		return c.JSON(http.StatusNoContent, fault) //204 is returned, indicating connection successful but no data
	}

	return c.JSON(http.StatusOK, tplQ)

}

func adminGetUsers(c echo.Context) error {
	//Get db from context, convert from interface to string
	cta := fmt.Sprintf("%v", c.Request().Context().Value("sub"))
	r := cache[cta].role
	db := dbase{"_system"}

	var bind string
	query := "FOR d in users RETURN {'email': d.email, 'role': d.role}"

	if r == "admin" {
		//Run query and response
		userQ, err := db.getQueries(query, bind, bind)

		//Catch error from the query
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		if userQ == nil {
			fault := "No data returned"
			return c.JSON(http.StatusBadRequest, fault)
		}

		return c.JSON(http.StatusOK, userQ)

	}

	return echo.ErrUnauthorized

}

func trendGetItem(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	it := "Items/" + id

	//Query to retrieve all ShoppingLists
	query_sl := "FOR s in ShoppingLists SORT s.date DESC RETURN {'list': s.name, 'date': s.date}"
	var bind string
	trQ, err := db.getQueries(query_sl, bind, bind)

	//Catch error from the query
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Query and map price trend
	var trend []d
	trI := 0

	//Test if trQ is empty.
	if len(trQ) == 0 {
		return c.JSON(http.StatusOK, "No trend data available")
	}

	dbx, ctx := aranDB(ah, db.db)
	query_tr := "FOR v, e IN 1..1 INBOUND @item @sl RETURN {'shop': v.name, 'branch': v.branch, 'city': v.city, 'country': v.country, 'currency': e.currency, 'price': e.price, 'date': e.date, 'special': e.special, 'list_id': e._id}"
	for _, trR := range trQ {
		var tres []d
		sl := trR["list"]
		b := d{"item": it, "sl": sl}

		if ct {
			data := aranQuery{query_tr, b, dbx, ctx}
			tres = data.aranQ()
		} else {
			fmt.Println("Failed to connect. Troubleshoot connection to ", ah)
			return c.JSON(http.StatusInternalServerError, err)
		}

		//If trend result is valid (not ""), then add to trend slice, increase counter
		if len(tres) > 0 {
			trend = append(trend, tres[0])
			trI++
		}

		//Only return 10 results
		if trI > 10 {
			break
		}
	}

	return c.JSON(http.StatusOK, trend)
}

/* ++++++++++++++
 * POST Functions
 * ++++++++++++++++
 */
//TODO: the postQueries methods can be tidied up to be less redundant/repeated

//c for collection name
func (i ItemNew) postQueries(c string, db dbase) (string, error) {

	var insertQ string
	dbx, ctx := aranDB(ah, db.db)

	if ct {
		data := aranInsertItem{c, i, dbx, ctx}
		insertQ = data.aranIns()
		fmt.Println("Meta key:", insertQ)
	} else {
		fmt.Println("Meta Key ", ah)
		var err = errors.New("failed to connect to db")
		return "", err
	}

	return insertQ, nil

}

func (i ShopNew) postQueries(c string, db dbase) (string, error) {

	var insertQ string
	dbx, ctx := aranDB(ah, db.db)

	if ct {
		data := aranInsertShop{c, i, dbx, ctx}
		insertQ = data.aranIns()
		fmt.Println("Meta key:", insertQ)
	} else {
		fmt.Println("Meta Key ", ah)
		var err = errors.New("failed to connect to db")
		return "", err
	}

	return insertQ, nil

}

func itemCreate(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	var data ItemNew
	var insertQ string
	coll := "Items"

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Nett <= 0 {
			return c.JSON(http.StatusBadRequest, "cannot have zero as nett")
		}
		if data.Brand == "" || data.Name == "" || data.Ntt_un == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		n := strings.ToLower(data.Name)
		b := strings.ToLower(data.Brand)
		data.Name, data.Brand = n, b

		//Run query and response
		insertQ, err = data.postQueries(coll, db)

		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

	}

	return c.JSON(http.StatusOK, insertQ)
}

func shopCreate(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	var data ShopNew
	var insertQ string
	coll := "Shops"

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Branch == "" || data.Name == "" || data.City == "" || data.Country == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		n := strings.ToLower(data.Name)
		b := strings.ToLower(data.Branch)
		ci := strings.ToLower(data.City)
		cy := strings.ToLower(data.Country)
		data.Name, data.Branch, data.City, data.Country = n, b, ci, cy

		//Run query and response
		insertQ, err = data.postQueries(coll, db)

		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

	}

	return c.JSON(http.StatusOK, "data submitted: "+insertQ)
}

//dbv - db name, returns data, edge name and error
func listCreateCore(dbv string) ([]d, string, error) {
	db := dbase{dbv}

	t := time.Now().Unix()
	n := "ShoppingList" + fmt.Sprint(t)
	e, err := db.edgeCreate(n)

	//Catch error from the query
	if err != nil {
		//Since Unix time will always be unqiue in this situation, any error means something has gone wrong server / code side
		return nil, "", errors.New("server error")
	}

	//Update ShoppingLists:
	query := "INSERT { name: @name, 'hidden': false, 'date': DATE_NOW(),} INTO ShoppingLists RETURN NEW"
	bind := "name"

	execQ, err := db.getQueries(query, bind, e.Name())

	if err != nil {
		return nil, "", errors.New("server error")
	}

	return execQ, e.Name(), nil
}

func listCreate(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))

	execQ, _, err := listCreateCore(dbv)

	//Catch errors
	if err != nil {
		if err.Error() == "id error" {
			return c.JSON(http.StatusBadRequest, "invalid id")
		} else if err.Error() == "server error" {
			return c.JSON(http.StatusInternalServerError, "server error")
		}
	}

	return c.JSON(http.StatusOK, execQ)

}

//Create ShoppingList from Template id
func listMake(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))

	//Get item id
	id := c.Param("id")

	//Retrieve Template based on id
	tpl, err := listTemplateDetailsCore(id, dbv)

	//Catch errors
	if err != nil {
		if err.Error() == "id error" {
			return c.JSON(http.StatusBadRequest, "invalid id")
		} else if err.Error() == "server error" {
			return c.JSON(http.StatusInternalServerError, "server error")
		}
	}

	if tpl == nil {
		fault := "No data to return."
		return c.JSON(http.StatusNoContent, fault) //204 is returned, indicating connection successful but no data
	}

	//Create new shopping list collection listCreateCore()
	_, edN, err := listCreateCore(dbv)

	//Catch errors
	if err != nil {
		if err.Error() == "id error" {
			return c.JSON(http.StatusBadRequest, "invalid id")
		} else if err.Error() == "server error" {
			return c.JSON(http.StatusInternalServerError, "server error")
		}
	}

	//Add items & shops to new shopping list
	// - Iterate Template tpl and enter into ShoppingList using ?
	for _, k := range tpl {
		u := k["items"]
		v, ok := u.([]interface{})
		if !ok {
			fmt.Println("Not OK!") //return error!!
		}

		for w := range v {
			u := v[w]
			r, ok := u.(map[string]interface{})
			if !ok {
				fmt.Println("Not OK2!") //return error!!
			}
			f := fmt.Sprintf("%v", r["shop_id"]) //From
			t := fmt.Sprintf("%v", r["item_id"]) //To
			q := float32(r["qty"].(float64))     //Qty
			shl := SlistEdge{t, f, 0, 0, "", false, false, q, ""}
			_, err = listAddItemCore(shl, dbv, edN) //Note that this function sets the date, hence 0 above.
			if err != nil {
				fmt.Println("Not OK3!") //return error!!
			}
		}

	}

	return c.JSON(http.StatusOK, "template created from shopping list")

}

func listEnableTemplates(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	_, err := db.colCreate("Templates")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, "Templates enabled")

}

func listCreateTemplate(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	t := time.Now().Unix()
	n := "Template" + fmt.Sprint(t)
	e, err := db.edgeCreate(n)

	//Catch error from the query
	if err != nil {
		//Since Unix time will always be unqiue in this situation, any error means something has gone wrong server / code side
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Update Templates:
	query := "INSERT { name: @name, 'hidden': false, 'date': DATE_NOW(),} INTO Templates RETURN NEW"
	bind := "name"

	execQ, err := db.getQueries(query, bind, e.Name())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, execQ)

}

func listMakeTemplate(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get ShoppingList id
	id := c.Param("id")

	//Retrieve ShoppingList based on id
	// - Only retrieve what is needed for a Template
	// - (not needed: price, currency, trolley, special)
	shQ, err := listGetShoppingCore(id, dbv, "qty")

	// - Catch errors
	if err != nil {

		if err.Error() == "id error" {
			return c.JSON(http.StatusBadRequest, "invalid id")
		} else if err.Error() == "server error" {
			return c.JSON(http.StatusInternalServerError, "server error")
		}
	}

	if shQ == nil {
		fault := "empty shopping list"
		return c.JSON(http.StatusNoContent, fault)
	}

	//Create Template collection - TODO: this is verbose. Create Core function. See listCreateTemplate

	t := time.Now().Unix()
	n := "Template" + fmt.Sprint(t)
	e, err := db.edgeCreate(n)

	// - Catch error from the query
	if err != nil {
		//Since Unix time will always be unqiue in this situation, any error means something has gone wrong server / code side
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Update Templates with reference to new Template:
	query := "INSERT { name: @name, 'hidden': false, 'date': DATE_NOW(),} INTO Templates RETURN NEW"
	bind := "name"

	_, err = db.getQueries(query, bind, e.Name())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Add entries to Template collection based on ShoppingList
	// - Iterate reduced ShoppingList shQ and enter into Template using addToTmpltCore()
	for _, k := range shQ {
		u := k["items"]
		v, ok := u.([]interface{})
		if !ok {
			fmt.Println("Not OK!") //return error!!
		}

		for w := range v {
			u := v[w]
			r, ok := u.(map[string]interface{})
			if !ok {
				fmt.Println("Not OK2!") //return error!!
			}
			f := fmt.Sprintf("%v", r["shop_id"]) //From
			t := fmt.Sprintf("%v", r["item_id"]) //To
			q := float32(r["qty"].(float64))     //Qty
			tdp := TplEdge{t, f, q}
			_, err = addToTmpltCore(tdp, dbv, e.Name())
			if err != nil {
				fmt.Println("Not OK3!") //return error!!
			}
		}

	}

	return c.JSON(http.StatusOK, "template created from shopping list")

}

func adminCreateUser(c echo.Context) error {
	//Get db from context, convert from interface to string
	cta := fmt.Sprintf("%v", c.Request().Context().Value("sub"))
	r := cache[cta].role
	db := dbase{"_system"}

	var data UserNew
	var dbnew string

	if r == "admin" {
		//DB is a random ID
		n := makeID()

		//Create DB
		dbn, err := dbCreate(ah, n)
		if err != nil {
			fmt.Println("Error creating new database", err)
		}

		fmt.Println("Database created, ", dbn)
		dbnew = dbn

		//Add to /users
		if err := c.Bind(&data); err != nil {
			return err
		} else if err == nil {

			query := "INSERT {'email': @email, 'db': '" + n + "', 'role': 'user'} INTO users"
			bind := "email"

			uQ, err := db.getQueries(query, bind, data.Email)

			if err != nil {
				return c.JSON(http.StatusInternalServerError, err)
			}

			fmt.Println(uQ)

		}

		//Create collections: Items, Shops, ShoppingLists
		//Temlates not created by default
		db2 := dbase{dbnew}
		_, err = db2.colCreate("Items")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		_, err = db2.colCreate("Shops")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		_, err = db2.colCreate("ShoppingLists")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		return c.JSON(http.StatusOK, "")
	}

	return echo.ErrUnauthorized

}

/*????????????
*
* PATCH
*
*?????????????
 */
func (d ItemNew) patchQueries(c, k string, db dbase) (string, error) {

	var upd string
	var err error

	dbx, ctx := aranDB(ah, db.db)

	n := strings.ToLower(d.Name)
	b := strings.ToLower(d.Brand)
	d.Name, d.Brand = n, b

	if ct {
		data := aranUpdateItem{c, k, d, dbx, ctx}
		upd, err = data.aranUp()

		if err != nil {
			return "", err
		}
	}

	return upd, nil

}

func (d ShopNew) patchQueries(c, k string, db dbase) (string, error) {

	var upd string
	var err error

	dbx, ctx := aranDB(ah, db.db)

	n := strings.ToLower(d.Name)
	b := strings.ToLower(d.Branch)
	ci := strings.ToLower(d.City)
	cy := strings.ToLower(d.Country)
	d.Name, d.Branch, d.City, d.Country = n, b, ci, cy

	if ct {
		data := aranUpdateShop{c, k, d, dbx, ctx}
		upd, err = data.aranUp()

		if err != nil {
			return "", err
		}
	}

	return upd, nil

}

func (d SlistEdgeItem) patchQueries(c, k string, db dbase) (string, error) {

	var upd string
	var err error

	dbx, ctx := aranDB(ah, db.db)

	if ct {
		data := aranUpdateSlist{c, k, d, dbx, ctx}
		upd, err = data.aranUp()

		if err != nil {
			return "", err
		}
	}

	return upd, nil

}

func (d ShopListsAll) patchQueries(c, k string, db dbase) (string, error) {

	var upd string
	var err error

	dbx, ctx := aranDB(ah, db.db)

	if ct {
		data := aranUpdateSlistAll{c, k, d, dbx, ctx}
		upd, err = data.aranUp()

		if err != nil {
			return "", err
		}
	}

	return upd, nil

}

func (d TplEdgeItem) patchQueries(c, k string, db dbase) (string, error) {

	var upd string
	var err error

	dbx, ctx := aranDB(ah, db.db)

	if ct {
		data := aranUpdateTpl{c, k, d, dbx, ctx}
		upd, err = data.aranUp()

		if err != nil {
			return "", err
		}
	}

	return upd, nil

}

func itemEdit(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	docKey := c.Param("id")
	col := "Items"

	var data ItemNew
	var update string

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Nett <= 0 {
			return c.JSON(http.StatusBadRequest, "cannot have zero as nett")
		}
		if data.Brand == "" || data.Name == "" || data.Ntt_un == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		update, err = data.patchQueries(col, docKey, db)

		if err != nil {
			//Since data was verified, any error is likely server related?
			return c.JSON(http.StatusInternalServerError, err)
		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)

}

func shopEdit(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	docKey := c.Param("id")
	col := "Shops"

	var data ShopNew
	var update string

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Branch == "" || data.Name == "" || data.City == "" || data.Country == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		update, err = data.patchQueries(col, docKey, db)

		if err != nil {
			//Since data was verified, any error is likely server related?
			return c.JSON(http.StatusInternalServerError, err)
		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)
}

func listSetHidden(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	col := "ShoppingLists"

	var data ShopListsAll
	var update string

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Date == 0 || data.Name == "" || data.Id == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		update, err = data.patchQueries(col, id, db)

		if err != nil {
			//Since data was verified, any error is likely server related?
			return c.JSON(http.StatusInternalServerError, err)
		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)

}

func listEdit(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	col := "ShoppingLists"

	var data ShopListsAll
	var update string

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Date == 0 || data.Name == "" || data.Id == "" || data.Label == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		update, err = data.patchQueries(col, id, db)

		if err != nil {
			//Since data was verified, any error is likely server related?
			return c.JSON(http.StatusInternalServerError, err)
		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)

}

//TODO: identical to above func. Tidy up!!
func listTemplateEdit(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	col := "Templates"

	var data ShopListsAll
	var update string

	if err := c.Bind(&data); err != nil {
		return err
	} else if err == nil {

		//Verify data, because Arango does not by default
		if data.Date == 0 || data.Name == "" || data.Id == "" || data.Label == "" {
			return c.JSON(http.StatusBadRequest, "all options must be set")
		}

		update, err = data.patchQueries(col, id, db)

		if err != nil {
			//Since data was verified, any error is likely server related?
			return c.JSON(http.StatusInternalServerError, err)
		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)

}

//Can update edge doc contents, but not change _from and _to
//Updates shopping lists qty, price, trolley, etc
func listSetTrolley(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	key := c.Param("key")
	fmt.Println("id:", id)
	fmt.Println("Key:", key)

	d := time.Now().Unix()
	var trolley SlistEdgeItem
	var update string

	s, err := db.getShoppingList(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	if err := c.Bind(&trolley); err != nil {
		return err
	} else if err == nil {

		trolley.Date = d
		update, err = trolley.patchQueries(s, key, db)

		if err != nil {
			//Since data was verified, any error is likely server related?
			return c.JSON(http.StatusInternalServerError, err)
		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)
}

//c = collection name
func listAddItemCore(s SlistEdge, dbv, c string) (string, error) {
	db := dbase{dbv}

	s.Date = time.Now().Unix()
	s.From = "Shops/" + s.From
	s.To = "Items/" + s.To

	dbx, ctx := aranDB(ah, db.db)
	col, err := dbx.Collection(ctx, c)
	if err != nil {
		return "", errors.New("server error")
	}

	meta, err := col.CreateDocument(ctx, s)
	if err != nil {
		return "", errors.New("server error")
	}

	return meta.Key, nil

}

func listAddItem(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")

	var sledge SlistEdge
	var ins string

	//Use id to retrieve ShoppingList name from ShoppingLists
	s, err := db.getShoppingList(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	//Add Edge
	//let d = DATE_NOW() INSERT { _to: "Items/382", _from: "Shops/246", date: d, price: 80.20, currency: "NAD", special: false, trolley: false, qty: 6, tag: ""} INTO ShoppingList20220717001 RETURN NEW
	if err := c.Bind(&sledge); err != nil {
		return err
	} else if err == nil {

		ins, err = listAddItemCore(sledge, dbv, s)

		if err != nil {
			if err.Error() == "server error" {
				return c.JSON(http.StatusInternalServerError, "server error")
			}
		}

	}

	return c.JSON(http.StatusOK, ins)
}

func listMoveItem(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	key := c.Param("key")

	//Use id to retrieve ShoppingList name from ShoppingLists
	s, err := db.getShoppingList(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	//Bind body: the _from should contain the id of the new shop
	var sledge SlistEdge
	if err := c.Bind(&sledge); err != nil {
		return c.JSON(http.StatusBadRequest, "Move item: error binding")
	} else if err == nil {

		sledge.Date = time.Now().Unix()
		sledge.From = "Shops/" + sledge.From
		sledge.To = "Items/" + sledge.To
	}

	//Link to Edge Collection
	dbx, ctx := aranDB(ah, db.db)
	col, err := dbx.Collection(ctx, s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Delete old edge
	_, err = col.RemoveDocument(ctx, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Insert new edge
	//Already have db and correct collection
	meta, err := col.CreateDocument(ctx, sledge)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	new := meta.Key

	return c.JSON(http.StatusOK, new)

	//

}

//Edge data, database ref, collection name
func addToTmpltCore(t TplEdge, dbv, c string) (string, error) {

	t.From = "Shops/" + t.From
	t.To = "Items/" + t.To

	dbx, ctx := aranDB(ah, dbv)
	col, err := dbx.Collection(ctx, c)
	if err != nil {
		return "", errors.New("server error")
	}

	meta, err := col.CreateDocument(ctx, t)
	if err != nil {
		return "", errors.New("server error")
	}

	return meta.Key, nil

}

func listAddToTemplate(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")

	var tpledge TplEdge
	var ins string
	var err2 error

	//Use id to retrieve Template name from Templates
	s, err := db.getTemplate(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	//Add Edge
	//let d = DATE_NOW() INSERT { _to: "Items/382", _from: "Shops/246", qty: 6} INTO Template20220717001 RETURN NEW
	if err := c.Bind(&tpledge); err != nil {
		return err
	} else if err == nil {

		ins, err2 = addToTmpltCore(tpledge, db.db, s)

		if err2 != nil {
			if err2.Error() == "server error" {
				return c.JSON(http.StatusInternalServerError, "server error")
			}
		}

	}

	return c.JSON(http.StatusOK, ins)
}

func listTemplateMoveItem(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	key := c.Param("key")

	//Use id to retrieve ShoppingList name from ShoppingLists
	s, err := db.getTemplate(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	//Bind body: the _from should contain the id of the new shop
	var tpledge TplEdge
	if err := c.Bind(&tpledge); err != nil {
		return c.JSON(http.StatusBadRequest, "Move item: error binding")
	} else if err == nil {

		tpledge.From = "Shops/" + tpledge.From
		tpledge.To = "Items/" + tpledge.To
	}

	//Link to Edge Collection
	dbx, ctx := aranDB(ah, db.db)
	col, err := dbx.Collection(ctx, s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Delete old edge
	_, err = col.RemoveDocument(ctx, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	//Insert new edge
	//Already have db and correct collection
	meta, err := col.CreateDocument(ctx, tpledge)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	new := meta.Key

	return c.JSON(http.StatusOK, new)

	//
}

func listUpdateTplItem(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	key := c.Param("key")

	var tpi TplEdgeItem
	var update string

	s, err := db.getTemplate(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	if err := c.Bind(&tpi); err != nil {
		return err
	} else if err == nil {

		//Verify qty is more than 0
		if tpi.Qty != 0 {

			update, err = tpi.patchQueries(s, key, db)

			if err != nil {
				//Since data was verified, any error is likely server related?
				return c.JSON(http.StatusInternalServerError, err)
			}

		}

	}

	return c.JSON(http.StatusOK, "update successful: "+update)
}

/*DDDDDDDD
*
* DELETE
*
 DDDDDDD*/

//Deleting an item: this should also remove history, i.e. remove the edge docs associated
func itemDelete(c echo.Context) error {
	//Get item id
	id := c.Param("id")
	id = "Items/" + id

	//Get all edge docs associated
	//Uses an http query: https://www.arangodb.com/docs/stable/http/collection-getting.html#reads-all-collections

	//Loop through array, and delete Edge doc where _to is the item

	//Delete the item doc

	return c.JSON(http.StatusOK, "still developing this one... ")
}

func shopDelete(c echo.Context) error {
	//Get item id
	id := c.Param("id")
	id = "Shops/" + id

	//Get all edge docs associated
	//Uses an http query: https://www.arangodb.com/docs/stable/http/collection-getting.html#reads-all-collections

	//Loop through array, and delete Edge doc where _from is the shop

	//Delete the shop doc

	return c.JSON(http.StatusOK, "still developing this one... ")
}

//Removes edge document (shopping list item) from Edge Collection (ShoppngListxyz123)
func listItemRemove(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	key := c.Param("key")

	//Use id to retrieve ShoppingList name from ShoppingLists
	s, err := db.getShoppingList(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	dbx, ctx := aranDB(ah, db.db)
	col, err := dbx.Collection(ctx, s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	meta, err := col.RemoveDocument(ctx, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	rem := meta.Key

	return c.JSON(http.StatusOK, rem)
}

func listTemplateItemRemove(c echo.Context) error {
	//Get db from context, convert from interface to string
	dbv := fmt.Sprintf("%v", c.Request().Context().Value("db"))
	db := dbase{dbv}

	//Get item id
	id := c.Param("id")
	key := c.Param("key")

	//Use id to retrieve ShoppingList name from ShoppingLists
	s, err := db.getTemplate(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid id")
	}

	dbx, ctx := aranDB(ah, db.db)
	col, err := dbx.Collection(ctx, s)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	meta, err := col.RemoveDocument(ctx, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	rem := meta.Key

	return c.JSON(http.StatusOK, rem)
}

/* !!!!!!!!!!!!!!
 *      MAIN
 * !!!!!!!!!!!!!!
 */
func main() {

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, //Can be *. localhost != 127.0.0.1 when evaluated.
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.PATCH, echo.DELETE},
	}))

	//All API endpoints require JWT validation
	//		Breakdown to only certain endpoints by e.g.
	//			r1.Use(middleJWT) - all Group("/items") will require
	//			r1.POST("/x", func, middleJWT)
	e.Use(middleJWT)

	// Routes

	// Router 1 - ITEMS
	r1 := e.Group("/items", middleUser)
	r1.GET("/view/:id", itemGetSpecific)
	r1.GET("/all", itemGetAll)
	r1.GET("/like/:part", itemGetLike)
	r1.POST("/new", itemCreate)
	r1.PATCH("/update/:id", itemEdit)
	r1.DELETE("delete/:id", itemDelete)

	// Router 2 - SHOPS
	r2 := e.Group("/shops", middleUser)
	r2.GET("/view/:id", shopGetSpecific)
	r2.GET("/all", shopGetAll)
	r2.GET("/like/:part", shopGetLike)
	r2.POST("/new", shopCreate)
	r2.PATCH("/update/:id", shopEdit)
	r2.DELETE("delete/:id", shopDelete)

	//Router 3
	r3 := e.Group("/shoppinglist", middleUser)

	//Shopping List, Trolley
	r3.GET("/allvisible", listGetVisible)
	r3.GET("/all", listGetAll)
	r3.GET("/view/:id", listGetShopping)
	r3.GET("/trolley/:id/:key", listGetTrolley)
	r3.GET("/name/:id", listGetName)
	r3.POST("/new", listCreate)
	r3.POST("/make/:id", listMake) //new based on Template id
	r3.PATCH("/hide/:id", listSetHidden)
	r3.PATCH("/edit/:id", listEdit)
	r3.PATCH("/trolley/:id/:key", listSetTrolley)
	r3.PATCH("/additem/:id", listAddItem)
	r3.PATCH("/moveitem/:id/:key", listMoveItem)
	r3.DELETE("/delete/item/:id/:key", listItemRemove)

	//Shopping list Templates
	r3.GET("/templates", listGetTemplates)
	r3.GET("/templates/details/:id", listTemplateDetails)
	r3.GET("/templates/name/:id", listTemplateName)
	r3.POST("/templates", listCreateTemplate)
	r3.POST("/templates/:id", listMakeTemplate) //new based on ShoppingList id
	r3.POST("/templates/enable", listEnableTemplates)
	r3.PATCH("/templates/:id", listTemplateEdit)          //edit template - temnplate name, date
	r3.PATCH("/templates/details/:id", listAddToTemplate) //edit template - add item to template
	r3.PATCH("/templates/moveitem/:id/:key", listTemplateMoveItem)
	r3.PATCH("/templates/details/:id/:key", listUpdateTplItem)       //edit template - edit individual item within template
	r3.DELETE("/templates/details/:id/:key", listTemplateItemRemove) //edit template - remove item from template
	/*

		r3.DELETE("/templates/:id", listRemoveTemplate) 	//delete template...or rather hide? Delete edge and enrty in Templates

	*/

	//Router 4 - SHOPPINGlist, Trolley
	r4 := e.Group("/trend", middleUser)
	r4.GET("/item/:id", trendGetItem)

	//Each method here must verify cache[sub].role == admin !!!!!
	r6 := e.Group("/admin", middleAdmin)
	r6.GET("/maybe", adminMaybe)
	r6.GET("/users", adminGetUsers)
	r6.POST("/users", adminCreateUser)
	//DELETE user (drop DB, remove from _system/users)
	//Setting as admin currently only possible by logging into container and running this query:

	e.Logger.Fatal(e.Start(":4000"))

}
