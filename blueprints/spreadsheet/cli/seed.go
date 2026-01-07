package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/spreadsheet/app/web"
	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
)

// NewSeed creates the seed command
func NewSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with demo data",
		Long: `Seed the Spreadsheet database with demo data for testing.

Creates a complete spreadsheet environment:
  - 3 users (alice, bob, charlie)
  - Sample workbook with multiple sheets
  - Sales data with formulas and charts
  - Formatted cells with conditional formatting

To reset the database, delete the data directory first:
  rm -rf ~/data/blueprint/spreadsheet && spreadsheet seed

Examples:
  spreadsheet seed                     # Seed with demo data
  spreadsheet seed --data /path/to    # Seed specific database`,
		RunE: runSeed,
	}

	return cmd
}

// seedUsers holds the test users data
var seedUsers = []struct {
	Email    string
	Name     string
	Password string
}{
	{"alice@example.com", "Alice Johnson", "password123"},
	{"bob@example.com", "Bob Smith", "password123"},
	{"charlie@example.com", "Charlie Brown", "password123"},
}

func runSeed(cmd *cobra.Command, args []string) error {
	Blank()
	Header("", "Seed Database")
	Blank()

	Summary("Data", dataDir)
	Blank()

	start := time.Now()
	stop := StartSpinner("Seeding database...")

	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: dataDir,
		Dev:     false,
	})
	if err != nil {
		stop()
		Error(fmt.Sprintf("Failed to create server: %v", err))
		return err
	}
	defer srv.Close()

	ctx := context.Background()

	// Create all test users
	createdUsers := make([]*users.User, 0, len(seedUsers))
	for _, u := range seedUsers {
		user, _, err := srv.UserService().Register(ctx, &users.RegisterIn{
			Email:    u.Email,
			Name:     u.Name,
			Password: u.Password,
		})
		if err != nil {
			// Try to get existing user
			user, _ = srv.UserService().GetByEmail(ctx, u.Email)
		}
		if user != nil {
			createdUsers = append(createdUsers, user)
		}
	}

	if len(createdUsers) == 0 {
		stop()
		return fmt.Errorf("failed to create any users")
	}

	ownerUser := createdUsers[0] // Alice is the owner

	// Create a sample workbook
	wb, err := srv.WorkbookService().Create(ctx, &workbooks.CreateIn{
		Name:      "Sales Report 2025",
		OwnerID:   ownerUser.ID,
		CreatedBy: ownerUser.ID,
	})
	if err != nil {
		stop()
		return fmt.Errorf("failed to create workbook: %v", err)
	}

	// Create sheets
	sheet1, err := srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Sales Data",
		Index:      0,
		CreatedBy:  ownerUser.ID,
	})
	if err != nil {
		stop()
		return fmt.Errorf("failed to create sheet: %v", err)
	}

	sheet2, _ := srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Summary",
		Index:      1,
		CreatedBy:  ownerUser.ID,
	})

	sheet3, _ := srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Formulas Demo",
		Index:      2,
		Color:      "#10B981",
		CreatedBy:  ownerUser.ID,
	})

	sheet4, _ := srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Inventory",
		Index:      3,
		Color:      "#F59E0B",
		CreatedBy:  ownerUser.ID,
	})

	// Create sample data in Sheet1 - Sales Data
	salesData := [][]interface{}{
		{"Product", "Q1 Sales", "Q2 Sales", "Q3 Sales", "Q4 Sales", "Total", "% of Total"},
		{"Laptops", 15000, 18000, 22000, 25000, "=SUM(B2:E2)", "=F2/$F$8"},
		{"Smartphones", 25000, 28000, 32000, 35000, "=SUM(B3:E3)", "=F3/$F$8"},
		{"Tablets", 8000, 9500, 11000, 12500, "=SUM(B4:E4)", "=F4/$F$8"},
		{"Accessories", 5000, 6000, 7500, 8500, "=SUM(B5:E5)", "=F5/$F$8"},
		{"Software", 12000, 14000, 16000, 18000, "=SUM(B6:E6)", "=F6/$F$8"},
		{"Services", 20000, 22000, 25000, 28000, "=SUM(B7:E7)", "=F7/$F$8"},
		{"Total", "=SUM(B2:B7)", "=SUM(C2:C7)", "=SUM(D2:D7)", "=SUM(E2:E7)", "=SUM(F2:F7)", "100%"},
	}

	for row, rowData := range salesData {
		for col, value := range rowData {
			var formula string
			var cellValue interface{}

			if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
				formula = strVal
			} else {
				cellValue = value
			}

			srv.CellService().Set(ctx, sheet1.ID, row, col, &cells.SetCellIn{
				Value:   cellValue,
				Formula: formula,
			})
		}
	}

	// Apply formatting to header row
	for col := 0; col < 7; col++ {
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet1.ID,
			Row:     0,
			Col:     col,
			Format: cells.Format{
				Bold:            true,
				BackgroundColor: "#1F2937",
				FontColor:       "#FFFFFF",
				HAlign:          "center",
			},
		})
	}

	// Apply currency format to numeric cells
	for row := 1; row < 8; row++ {
		for col := 1; col < 6; col++ {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet1.ID,
				Row:     row,
				Col:     col,
				Format: cells.Format{
					NumberFormat: "$#,##0",
					HAlign:       "right",
				},
			})
		}
	}

	// Apply percentage format to last column
	for row := 1; row < 8; row++ {
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet1.ID,
			Row:     row,
			Col:     6,
			Format: cells.Format{
				NumberFormat: "0.0%",
				HAlign:       "right",
			},
		})
	}

	// Create summary sheet data
	if sheet2 != nil {
		summaryData := [][]interface{}{
			{"Key Metrics", "Value"},
			{"Total Revenue", "='Sales Data'!F8"},
			{"Best Product", "Smartphones"},
			{"Best Quarter", "Q4"},
			{"YoY Growth", "15.2%"},
			{"", ""},
			{"Quarterly Breakdown", ""},
			{"Q1", "='Sales Data'!B8"},
			{"Q2", "='Sales Data'!C8"},
			{"Q3", "='Sales Data'!D8"},
			{"Q4", "='Sales Data'!E8"},
		}

		for row, rowData := range summaryData {
			for col, value := range rowData {
				var formula string
				var cellValue interface{}

				if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
					formula = strVal
				} else {
					cellValue = value
				}

				srv.CellService().Set(ctx, sheet2.ID, row, col, &cells.SetCellIn{
					Value:   cellValue,
					Formula: formula,
				})
			}
		}
	}

	// Create Formulas Demo sheet data
	if sheet3 != nil {
		formulasData := [][]interface{}{
			{"Formula Type", "Example", "Result"},
			{"", "", ""},
			{"Math Functions", "", ""},
			{"SUM", "=SUM(10,20,30)", "=SUM(10,20,30)"},
			{"AVERAGE", "=AVERAGE(10,20,30)", "=AVERAGE(10,20,30)"},
			{"MAX/MIN", "=MAX(10,20,30)", "=MAX(10,20,30)"},
			{"ROUND", "=ROUND(3.14159,2)", "=ROUND(3.14159,2)"},
			{"SQRT", "=SQRT(144)", "=SQRT(144)"},
			{"POWER", "=POWER(2,10)", "=POWER(2,10)"},
			{"", "", ""},
			{"Text Functions", "", ""},
			{"CONCATENATE", `=CONCATENATE("Hello"," ","World")`, `=CONCATENATE("Hello"," ","World")`},
			{"UPPER", `=UPPER("hello")`, `=UPPER("hello")`},
			{"LEN", `=LEN("Spreadsheet")`, `=LEN("Spreadsheet")`},
			{"LEFT", `=LEFT("Hello",3)`, `=LEFT("Hello",3)`},
			{"MID", `=MID("Spreadsheet",7,5)`, `=MID("Spreadsheet",7,5)`},
			{"", "", ""},
			{"Logical Functions", "", ""},
			{"IF", "=IF(10>5,\"Yes\",\"No\")", "=IF(10>5,\"Yes\",\"No\")"},
			{"AND", "=AND(TRUE,TRUE)", "=AND(TRUE,TRUE)"},
			{"OR", "=OR(TRUE,FALSE)", "=OR(TRUE,FALSE)"},
			{"NOT", "=NOT(FALSE)", "=NOT(FALSE)"},
			{"", "", ""},
			{"Statistical Functions", "", ""},
			{"COUNT", "=COUNT(1,2,3,\"a\")", "=COUNT(1,2,3,\"a\")"},
			{"COUNTA", "=COUNTA(1,2,3,\"a\")", "=COUNTA(1,2,3,\"a\")"},
			{"MEDIAN", "=MEDIAN(1,2,3,4,5)", "=MEDIAN(1,2,3,4,5)"},
			{"STDEV", "=STDEV(1,2,3,4,5)", "=STDEV(1,2,3,4,5)"},
			{"", "", ""},
			{"Date Functions", "", ""},
			{"TODAY", "=TODAY()", "=TODAY()"},
			{"NOW", "=NOW()", "=NOW()"},
			{"YEAR", "=YEAR(TODAY())", "=YEAR(TODAY())"},
			{"MONTH", "=MONTH(TODAY())", "=MONTH(TODAY())"},
			{"", "", ""},
			{"Cross-Sheet References", "", ""},
			{"Total Revenue", "='Sales Data'!F8", "='Sales Data'!F8"},
			{"Q1 Sales", "='Sales Data'!B8", "='Sales Data'!B8"},
		}

		for row, rowData := range formulasData {
			for col, value := range rowData {
				var formula string
				var cellValue interface{}

				if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
					formula = strVal
				} else {
					cellValue = value
				}

				srv.CellService().Set(ctx, sheet3.ID, row, col, &cells.SetCellIn{
					Value:   cellValue,
					Formula: formula,
				})
			}
		}

		// Apply formatting to formula headers
		headerRows := []int{0, 2, 10, 16, 22, 28, 34}
		for _, row := range headerRows {
			for col := 0; col < 3; col++ {
				srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
					SheetID: sheet3.ID,
					Row:     row,
					Col:     col,
					Format: cells.Format{
						Bold:            true,
						BackgroundColor: "#E5E7EB",
					},
				})
			}
		}

		srv.SheetService().SetColWidth(ctx, sheet3.ID, 0, 150)
		srv.SheetService().SetColWidth(ctx, sheet3.ID, 1, 250)
		srv.SheetService().SetColWidth(ctx, sheet3.ID, 2, 150)
	}

	// Create Inventory sheet with lookup data
	if sheet4 != nil {
		inventoryData := [][]interface{}{
			{"SKU", "Product Name", "Category", "Price", "Stock", "Status", "Value"},
			{"SKU001", "Laptop Pro 15", "Electronics", 1299.99, 45, "=IF(E2<10,\"Low Stock\",IF(E2<25,\"Medium\",\"In Stock\"))", "=D2*E2"},
			{"SKU002", "Wireless Mouse", "Accessories", 29.99, 150, "=IF(E3<10,\"Low Stock\",IF(E3<25,\"Medium\",\"In Stock\"))", "=D3*E3"},
			{"SKU003", "USB-C Hub", "Accessories", 59.99, 8, "=IF(E4<10,\"Low Stock\",IF(E4<25,\"Medium\",\"In Stock\"))", "=D4*E4"},
			{"SKU004", "Monitor 27\"", "Electronics", 449.99, 22, "=IF(E5<10,\"Low Stock\",IF(E5<25,\"Medium\",\"In Stock\"))", "=D5*E5"},
			{"SKU005", "Keyboard Mech", "Accessories", 89.99, 65, "=IF(E6<10,\"Low Stock\",IF(E6<25,\"Medium\",\"In Stock\"))", "=D6*E6"},
			{"SKU006", "Webcam HD", "Electronics", 79.99, 5, "=IF(E7<10,\"Low Stock\",IF(E7<25,\"Medium\",\"In Stock\"))", "=D7*E7"},
			{"SKU007", "Headphones BT", "Audio", 149.99, 35, "=IF(E8<10,\"Low Stock\",IF(E8<25,\"Medium\",\"In Stock\"))", "=D8*E8"},
			{"SKU008", "Speakers 2.1", "Audio", 199.99, 18, "=IF(E9<10,\"Low Stock\",IF(E9<25,\"Medium\",\"In Stock\"))", "=D9*E9"},
			{"", "", "", "", "", "", ""},
			{"", "", "Totals:", "=AVERAGE(D2:D9)", "=SUM(E2:E9)", "", "=SUM(G2:G9)"},
			{"", "", "", "", "", "", ""},
			{"", "Lookup Demo", "", "", "", "", ""},
			{"", "Enter SKU:", "SKU003", "", "", "", ""},
			{"", "Product:", "=VLOOKUP(C14,A2:G9,2,FALSE)", "", "", "", ""},
			{"", "Price:", "=VLOOKUP(C14,A2:G9,4,FALSE)", "", "", "", ""},
			{"", "Stock:", "=VLOOKUP(C14,A2:G9,5,FALSE)", "", "", "", ""},
		}

		for row, rowData := range inventoryData {
			for col, value := range rowData {
				var formula string
				var cellValue interface{}

				if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
					formula = strVal
				} else {
					cellValue = value
				}

				srv.CellService().Set(ctx, sheet4.ID, row, col, &cells.SetCellIn{
					Value:   cellValue,
					Formula: formula,
				})
			}
		}

		// Apply formatting to header
		for col := 0; col < 7; col++ {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet4.ID,
				Row:     0,
				Col:     col,
				Format: cells.Format{
					Bold:            true,
					BackgroundColor: "#F59E0B",
					FontColor:       "#FFFFFF",
					HAlign:          "center",
				},
			})
		}

		// Price format
		for row := 1; row < 11; row++ {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet4.ID,
				Row:     row,
				Col:     3,
				Format: cells.Format{
					NumberFormat: "$#,##0.00",
					HAlign:       "right",
				},
			})
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet4.ID,
				Row:     row,
				Col:     6,
				Format: cells.Format{
					NumberFormat: "$#,##0.00",
					HAlign:       "right",
				},
			})
		}

		// Lookup section formatting
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet4.ID,
			Row:     12,
			Col:     1,
			Format: cells.Format{
				Bold:            true,
				BackgroundColor: "#FEF3C7",
			},
		})

		srv.SheetService().SetColWidth(ctx, sheet4.ID, 0, 80)
		srv.SheetService().SetColWidth(ctx, sheet4.ID, 1, 140)
		srv.SheetService().SetColWidth(ctx, sheet4.ID, 2, 100)
		srv.SheetService().SetColWidth(ctx, sheet4.ID, 3, 80)
		srv.SheetService().SetColWidth(ctx, sheet4.ID, 4, 60)
		srv.SheetService().SetColWidth(ctx, sheet4.ID, 5, 80)
		srv.SheetService().SetColWidth(ctx, sheet4.ID, 6, 100)
	}

	// Set column widths for better display
	srv.SheetService().SetColWidth(ctx, sheet1.ID, 0, 120) // Product column
	for col := 1; col < 7; col++ {
		srv.SheetService().SetColWidth(ctx, sheet1.ID, col, 100)
	}

	stop()
	Step("", "Database seeded", time.Since(start))
	Blank()
	Success("Sample data created")
	Blank()

	Summary(
		"Users", fmt.Sprintf("%d users (alice, bob, charlie)", len(createdUsers)),
		"Password", "password123",
		"Workbook", wb.Name,
		"Sheets", "Sales Data, Summary, Formulas Demo, Inventory",
	)
	Blank()
	Hint("Start the server with: spreadsheet serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && spreadsheet seed")
	Blank()

	return nil
}
