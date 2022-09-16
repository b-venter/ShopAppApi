package main

/**
Compare all databases ShoppingLIstYYYYMMDD123 with ShoppingLists entries
Can be used to determine if shopping lists missing from ShoppingLists
	dbx, ctx := aranDB(ah, adb)

	found, err := dbx.Collections(ctx)

	if err != nil {
		fmt.Println("Error getting collection: ", err)
	}

	for i, a := range found {
		b = a.Name()

		aranQuery{"For doc in ShoppingList FILTER doc.name === b RETURN true"}
	}
*/

/**
Create Items, ShoppingLists and Shops collections initially
*/
