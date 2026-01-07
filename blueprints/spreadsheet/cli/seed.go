package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/spreadsheet/app/web"
	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/charts"
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

	sheet5, _ := srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Formats",
		Index:      4,
		Color:      "#EC4899",
		CreatedBy:  ownerUser.ID,
	})

	sheet6, _ := srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Charts",
		Index:      5,
		Color:      "#8B5CF6",
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

	// Create Formats demo sheet with comprehensive Google Sheets-compatible formatting
	if sheet5 != nil {
		// ============================================================
		// FORMATS SHEET - Comprehensive Formatting Reference
		// All formats are 100% compatible with Google Spreadsheets
		// ============================================================

		formatsData := [][]interface{}{
			// Section 1: Font Styles (Rows 0-11)
			{"FONT STYLES", "", "", "", "", ""},
			{"Style", "Example Text", "", "", "", ""},
			{"Bold", "Bold Text", "", "", "", ""},
			{"Italic", "Italic Text", "", "", "", ""},
			{"Underline", "Underlined", "", "", "", ""},
			{"Strikethrough", "Crossed Out", "", "", "", ""},
			{"Bold + Italic", "Bold Italic", "", "", "", ""},
			{"All Styles", "All Combined", "", "", "", ""},
			{"", "", "", "", "", ""},
			{"FONT SIZES", "", "", "", "", ""},
			{"8pt", "Small", "10pt", "Normal", "12pt", "Medium"},
			{"14pt", "Large", "18pt", "Larger", "24pt", "Headline"},
			{"", "", "", "", "", ""},
			// Section 2: Font Colors (Rows 13-19)
			{"FONT COLORS", "", "", "", "", ""},
			{"Black", "Red", "Green", "Blue", "Orange", "Purple"},
			{"#000000", "#FF0000", "#00FF00", "#0000FF", "#FF9900", "#9900FF"},
			{"Dark Gray", "Dark Red", "Dark Green", "Dark Blue", "Brown", "Magenta"},
			{"#666666", "#990000", "#006600", "#000099", "#996633", "#FF00FF"},
			{"", "", "", "", "", ""},
			// Section 3: Background Colors (Rows 19-25)
			{"BACKGROUND COLORS", "", "", "", "", ""},
			{"Light Red", "Light Green", "Light Blue", "Light Yellow", "Light Orange", "Light Purple"},
			{"#FFCDD2", "#C8E6C9", "#BBDEFB", "#FFF9C4", "#FFE0B2", "#E1BEE7"},
			{"Red", "Green", "Blue", "Yellow", "Orange", "Purple"},
			{"#F44336", "#4CAF50", "#2196F3", "#FFEB3B", "#FF9800", "#9C27B0"},
			{"", "", "", "", "", ""},
			// Section 4: Horizontal Alignment (Rows 25-29)
			{"HORIZONTAL ALIGNMENT", "", "", "", "", ""},
			{"Left Aligned", "", "", "", "", ""},
			{"Center Aligned", "", "", "", "", ""},
			{"Right Aligned", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			// Section 5: Vertical Alignment (Rows 30-34)
			{"VERTICAL ALIGNMENT", "", "", "", "", ""},
			{"Top", "Middle", "Bottom", "", "", ""},
			{"", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			// Section 6: Text Rotation (Rows 35-39)
			{"TEXT ROTATION", "", "", "", "", ""},
			{"0°", "45°", "90°", "-45°", "-90°", ""},
			{"Normal", "Diagonal", "Vertical", "Reverse", "Down", ""},
			{"", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			// Section 7: Indentation (Rows 40-44)
			{"TEXT INDENTATION", "", "", "", "", ""},
			{"No indent", "", "", "", "", ""},
			{"Indent 1", "", "", "", "", ""},
			{"Indent 2", "", "", "", "", ""},
			{"Indent 3", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			// Section 8: Text Wrapping (Rows 45-48)
			{"TEXT WRAPPING", "", "", "", "", ""},
			{"This is a long text that demonstrates text wrapping in cells", "", "", "", "", ""},
			{"No wrap - text extends beyond cell", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			// Section 9: Border Styles (Rows 49-58)
			{"BORDER STYLES", "", "", "", "", ""},
			{"Thin", "Medium", "Thick", "Dotted", "Dashed", "Double"},
			{"", "", "", "", "", ""},
			{"", "", "", "", "", ""},
			{"Border Colors:", "", "", "", "", ""},
			{"Black Border", "Red Border", "Blue Border", "Green Border", "Orange Border", ""},
			{"", "", "", "", "", ""},
			{"Box Borders:", "", "", "", "", ""},
			{"Full Box", "Top+Bottom", "Left+Right", "Top Only", "Custom", ""},
			{"", "", "", "", "", ""},
			// Section 10: Number Formats (Rows 59-78)
			{"NUMBER FORMATS", "", "", "", "", ""},
			{"Format Type", "Value", "Formatted", "Pattern", "", ""},
			{"Integer", 1234567, 1234567, "#,##0", "", ""},
			{"Decimal (2)", 1234.567, 1234.567, "#,##0.00", "", ""},
			{"Decimal (4)", 3.14159265, 3.14159265, "#,##0.0000", "", ""},
			{"", "", "", "", "", ""},
			{"Currency (USD)", 1234.50, 1234.50, "$#,##0.00", "", ""},
			{"Currency (neg)", -500.00, -500.00, "$#,##0.00", "", ""},
			{"Accounting", 1234.50, 1234.50, `"$"#,##0.00`, "", ""},
			{"", "", "", "", "", ""},
			{"Percentage", 0.1525, 0.1525, "0.0%", "", ""},
			{"Percentage (2)", 0.85678, 0.85678, "0.00%", "", ""},
			{"", "", "", "", "", ""},
			{"Date (ISO)", "2025-01-15", "2025-01-15", "yyyy-mm-dd", "", ""},
			{"Date (US)", "2025-01-15", "2025-01-15", "mm/dd/yyyy", "", ""},
			{"Date (Long)", "2025-01-15", "2025-01-15", "mmmm d, yyyy", "", ""},
			{"Time", "14:30:00", "14:30:00", "hh:mm:ss", "", ""},
			{"DateTime", "2025-01-15 14:30", "2025-01-15 14:30", "yyyy-mm-dd hh:mm", "", ""},
			{"", "", "", "", "", ""},
			{"Scientific", 123456789, 123456789, "0.00E+00", "", ""},
			{"Fraction", 0.5, 0.5, "# ?/?", "", ""},
			{"Custom", 12345, 12345, `#,##0" units"`, "", ""},
			{"", "", "", "", "", ""},
			// Section 11: Combined Formatting Examples (Rows 82-92)
			{"COMBINED FORMATTING EXAMPLES", "", "", "", "", ""},
			{"Header Style", "Important", "Warning", "Error", "Success", "Info"},
			{"", "", "", "", "", ""},
			{"Financial Report Header", "", "", "", "", ""},
			{"Revenue", "$125,000", "", "Cost", "$98,000", ""},
			{"Profit", "$27,000", "", "Margin", "21.6%", ""},
			{"", "", "", "", "", ""},
			{"Status Indicators", "", "", "", "", ""},
			{"Active", "Pending", "Completed", "Cancelled", "On Hold", ""},
			{"", "", "", "", "", ""},
			// Section 12: Font Families (Rows 92-98)
			{"FONT FAMILIES (Google Sheets Compatible)", "", "", "", "", ""},
			{"Arial", "Helvetica", "Times", "Georgia", "Courier", "Verdana"},
			{"Default", "Clean", "Serif", "Elegant", "Mono", "Readable"},
			{"", "", "", "", "", ""},
			{"", "", "", "", "", ""},
		}

		// Populate cells
		for row, rowData := range formatsData {
			for col, value := range rowData {
				if value == nil || value == "" {
					continue
				}
				srv.CellService().Set(ctx, sheet5.ID, row, col, &cells.SetCellIn{
					Value: value,
				})
			}
		}

		// ============================================================
		// APPLY FORMATTING
		// ============================================================

		// --- Section Headers Formatting ---
		sectionHeaders := []int{0, 9, 13, 19, 25, 30, 35, 40, 45, 49, 59, 81, 92}
		for _, row := range sectionHeaders {
			for col := 0; col < 6; col++ {
				srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
					SheetID: sheet5.ID,
					Row:     row,
					Col:     col,
					Format: cells.Format{
						Bold:            true,
						FontSize:        14,
						FontColor:       "#FFFFFF",
						BackgroundColor: "#EC4899",
						HAlign:          "left",
					},
				})
			}
		}

		// --- Font Styles (Rows 2-7) ---
		// Bold
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 2, Col: 1,
			Format: cells.Format{Bold: true},
		})
		// Italic
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 3, Col: 1,
			Format: cells.Format{Italic: true},
		})
		// Underline
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 4, Col: 1,
			Format: cells.Format{Underline: true},
		})
		// Strikethrough
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 5, Col: 1,
			Format: cells.Format{Strikethrough: true},
		})
		// Bold + Italic
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 6, Col: 1,
			Format: cells.Format{Bold: true, Italic: true},
		})
		// All Combined
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 7, Col: 1,
			Format: cells.Format{Bold: true, Italic: true, Underline: true},
		})

		// --- Font Sizes (Rows 10-11) ---
		fontSizes := []struct{ row, col, size int }{
			{10, 0, 8}, {10, 1, 8}, {10, 2, 10}, {10, 3, 10}, {10, 4, 12}, {10, 5, 12},
			{11, 0, 14}, {11, 1, 14}, {11, 2, 18}, {11, 3, 18}, {11, 4, 24}, {11, 5, 24},
		}
		for _, fs := range fontSizes {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: fs.row, Col: fs.col,
				Format: cells.Format{FontSize: fs.size},
			})
		}

		// --- Font Colors (Rows 14-17) ---
		fontColors := []struct{ row, col int; color string }{
			{14, 0, "#000000"}, {14, 1, "#FF0000"}, {14, 2, "#00AA00"}, {14, 3, "#0000FF"}, {14, 4, "#FF9900"}, {14, 5, "#9900FF"},
			{15, 0, "#000000"}, {15, 1, "#FF0000"}, {15, 2, "#00AA00"}, {15, 3, "#0000FF"}, {15, 4, "#FF9900"}, {15, 5, "#9900FF"},
			{16, 0, "#666666"}, {16, 1, "#990000"}, {16, 2, "#006600"}, {16, 3, "#000099"}, {16, 4, "#996633"}, {16, 5, "#FF00FF"},
			{17, 0, "#666666"}, {17, 1, "#990000"}, {17, 2, "#006600"}, {17, 3, "#000099"}, {17, 4, "#996633"}, {17, 5, "#FF00FF"},
		}
		for _, fc := range fontColors {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: fc.row, Col: fc.col,
				Format: cells.Format{FontColor: fc.color, Bold: true},
			})
		}

		// --- Background Colors (Rows 20-23) ---
		bgColors := []struct{ row, col int; bg, fg string }{
			// Light colors
			{20, 0, "#FFCDD2", "#000000"}, {20, 1, "#C8E6C9", "#000000"}, {20, 2, "#BBDEFB", "#000000"},
			{20, 3, "#FFF9C4", "#000000"}, {20, 4, "#FFE0B2", "#000000"}, {20, 5, "#E1BEE7", "#000000"},
			{21, 0, "#FFCDD2", "#666666"}, {21, 1, "#C8E6C9", "#666666"}, {21, 2, "#BBDEFB", "#666666"},
			{21, 3, "#FFF9C4", "#666666"}, {21, 4, "#FFE0B2", "#666666"}, {21, 5, "#E1BEE7", "#666666"},
			// Dark colors
			{22, 0, "#F44336", "#FFFFFF"}, {22, 1, "#4CAF50", "#FFFFFF"}, {22, 2, "#2196F3", "#FFFFFF"},
			{22, 3, "#FFEB3B", "#000000"}, {22, 4, "#FF9800", "#FFFFFF"}, {22, 5, "#9C27B0", "#FFFFFF"},
			{23, 0, "#F44336", "#FFFFFF"}, {23, 1, "#4CAF50", "#FFFFFF"}, {23, 2, "#2196F3", "#FFFFFF"},
			{23, 3, "#FFEB3B", "#000000"}, {23, 4, "#FF9800", "#FFFFFF"}, {23, 5, "#9C27B0", "#FFFFFF"},
		}
		for _, bc := range bgColors {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: bc.row, Col: bc.col,
				Format: cells.Format{BackgroundColor: bc.bg, FontColor: bc.fg, HAlign: "center"},
			})
		}

		// --- Horizontal Alignment (Rows 26-28) ---
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 26, Col: 0,
			Format: cells.Format{HAlign: "left", BackgroundColor: "#F5F5F5"},
		})
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 27, Col: 0,
			Format: cells.Format{HAlign: "center", BackgroundColor: "#F5F5F5"},
		})
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 28, Col: 0,
			Format: cells.Format{HAlign: "right", BackgroundColor: "#F5F5F5"},
		})

		// --- Vertical Alignment (Row 31) with increased row height ---
		srv.SheetService().SetRowHeight(ctx, sheet5.ID, 31, 60)
		srv.SheetService().SetRowHeight(ctx, sheet5.ID, 32, 60)
		srv.SheetService().SetRowHeight(ctx, sheet5.ID, 33, 60)
		vAligns := []string{"top", "middle", "bottom"}
		for col, va := range vAligns {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 31, Col: col,
				Format: cells.Format{VAlign: va, BackgroundColor: "#E3F2FD", HAlign: "center"},
			})
		}

		// --- Text Rotation (Row 36-37) with increased row height ---
		srv.SheetService().SetRowHeight(ctx, sheet5.ID, 36, 80)
		srv.SheetService().SetRowHeight(ctx, sheet5.ID, 37, 80)
		rotations := []int{0, 45, 90, -45, -90}
		for col, rot := range rotations {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 36, Col: col,
				Format: cells.Format{TextRotation: rot, BackgroundColor: "#FFF3E0", HAlign: "center", VAlign: "middle"},
			})
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 37, Col: col,
				Format: cells.Format{TextRotation: rot, BackgroundColor: "#FFF3E0", HAlign: "center", VAlign: "middle"},
			})
		}

		// --- Text Indentation (Rows 41-44) ---
		indents := []int{0, 1, 2, 3}
		for i, indent := range indents {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 41 + i, Col: 0,
				Format: cells.Format{Indent: indent, BackgroundColor: "#F3E5F5"},
			})
		}

		// --- Text Wrapping (Rows 46-47) ---
		srv.SheetService().SetRowHeight(ctx, sheet5.ID, 46, 50)
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 46, Col: 0,
			Format: cells.Format{WrapText: true, BackgroundColor: "#E8F5E9"},
		})
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 47, Col: 0,
			Format: cells.Format{WrapText: false, BackgroundColor: "#FFEBEE"},
		})

		// --- Border Styles (Row 50) ---
		borderStyles := []string{"thin", "medium", "thick", "dotted", "dashed", "double"}
		for col, style := range borderStyles {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 50, Col: col,
				Format: cells.Format{
					BorderTop:    cells.Border{Style: style, Color: "#000000"},
					BorderRight:  cells.Border{Style: style, Color: "#000000"},
					BorderBottom: cells.Border{Style: style, Color: "#000000"},
					BorderLeft:   cells.Border{Style: style, Color: "#000000"},
					HAlign:       "center",
					BackgroundColor: "#FAFAFA",
				},
			})
		}

		// --- Border Colors (Row 54) ---
		borderColors := []string{"#000000", "#F44336", "#2196F3", "#4CAF50", "#FF9800"}
		for col, color := range borderColors {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 54, Col: col,
				Format: cells.Format{
					BorderTop:    cells.Border{Style: "medium", Color: color},
					BorderRight:  cells.Border{Style: "medium", Color: color},
					BorderBottom: cells.Border{Style: "medium", Color: color},
					BorderLeft:   cells.Border{Style: "medium", Color: color},
					HAlign:       "center",
				},
			})
		}

		// --- Box Border Variations (Row 57) ---
		// Full Box
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 57, Col: 0,
			Format: cells.Format{
				BorderTop: cells.Border{Style: "thick", Color: "#1976D2"},
				BorderRight: cells.Border{Style: "thick", Color: "#1976D2"},
				BorderBottom: cells.Border{Style: "thick", Color: "#1976D2"},
				BorderLeft: cells.Border{Style: "thick", Color: "#1976D2"},
				HAlign: "center", BackgroundColor: "#E3F2FD",
			},
		})
		// Top+Bottom only
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 57, Col: 1,
			Format: cells.Format{
				BorderTop: cells.Border{Style: "thick", Color: "#388E3C"},
				BorderBottom: cells.Border{Style: "thick", Color: "#388E3C"},
				HAlign: "center", BackgroundColor: "#E8F5E9",
			},
		})
		// Left+Right only
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 57, Col: 2,
			Format: cells.Format{
				BorderRight: cells.Border{Style: "thick", Color: "#F57C00"},
				BorderLeft: cells.Border{Style: "thick", Color: "#F57C00"},
				HAlign: "center", BackgroundColor: "#FFF3E0",
			},
		})
		// Top only
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 57, Col: 3,
			Format: cells.Format{
				BorderTop: cells.Border{Style: "thick", Color: "#7B1FA2"},
				HAlign: "center", BackgroundColor: "#F3E5F5",
			},
		})
		// Custom mixed
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 57, Col: 4,
			Format: cells.Format{
				BorderTop: cells.Border{Style: "double", Color: "#D32F2F"},
				BorderBottom: cells.Border{Style: "dashed", Color: "#1976D2"},
				BorderLeft: cells.Border{Style: "thin", Color: "#388E3C"},
				BorderRight: cells.Border{Style: "dotted", Color: "#F57C00"},
				HAlign: "center", BackgroundColor: "#FFF8E1",
			},
		})

		// --- Number Formats (Rows 61-79) ---
		numberFormats := []struct{ row int; format string }{
			{61, "#,##0"},
			{62, "#,##0.00"},
			{63, "#,##0.0000"},
			{65, "$#,##0.00"},
			{66, "$#,##0.00"},
			{67, "$#,##0.00"},
			{69, "0.0%"},
			{70, "0.00%"},
			{72, "yyyy-mm-dd"},
			{73, "mm/dd/yyyy"},
			{74, "mmmm d, yyyy"},
			{75, "hh:mm:ss"},
			{76, "yyyy-mm-dd hh:mm"},
			{78, "0.00E+00"},
			{79, "# ?/?"},
			{80, `#,##0" units"`},
		}
		for _, nf := range numberFormats {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: nf.row, Col: 2,
				Format: cells.Format{NumberFormat: nf.format, HAlign: "right"},
			})
		}

		// --- Combined Formatting Examples (Rows 82-90) ---
		// Header row styles
		headerStyles := []struct{ col int; bg, fg string }{
			{0, "#1F2937", "#FFFFFF"},
			{1, "#DC2626", "#FFFFFF"},
			{2, "#F59E0B", "#000000"},
			{3, "#EF4444", "#FFFFFF"},
			{4, "#10B981", "#FFFFFF"},
			{5, "#3B82F6", "#FFFFFF"},
		}
		for _, hs := range headerStyles {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 82, Col: hs.col,
				Format: cells.Format{
					Bold: true, FontSize: 12, BackgroundColor: hs.bg, FontColor: hs.fg, HAlign: "center",
					BorderTop: cells.Border{Style: "thin", Color: "#000000"},
					BorderBottom: cells.Border{Style: "thin", Color: "#000000"},
				},
			})
		}

		// Financial report header
		srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
			SheetID: sheet5.ID, Row: 84, Col: 0,
			Format: cells.Format{Bold: true, FontSize: 14, BackgroundColor: "#1F2937", FontColor: "#FFFFFF"},
		})
		// Financial data cells
		finCells := []struct{ row, col int; format string }{
			{85, 1, "$#,##0"}, {85, 4, "$#,##0"},
			{86, 1, "$#,##0"}, {86, 4, "0.0%"},
		}
		for _, fc := range finCells {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: fc.row, Col: fc.col,
				Format: cells.Format{NumberFormat: fc.format, HAlign: "right", Bold: true},
			})
		}

		// Status indicators
		statusStyles := []struct{ col int; bg, fg string }{
			{0, "#10B981", "#FFFFFF"}, // Active - Green
			{1, "#F59E0B", "#000000"}, // Pending - Amber
			{2, "#3B82F6", "#FFFFFF"}, // Completed - Blue
			{3, "#EF4444", "#FFFFFF"}, // Cancelled - Red
			{4, "#6B7280", "#FFFFFF"}, // On Hold - Gray
		}
		for _, ss := range statusStyles {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 89, Col: ss.col,
				Format: cells.Format{
					BackgroundColor: ss.bg, FontColor: ss.fg, Bold: true, HAlign: "center",
					BorderTop: cells.Border{Style: "thin", Color: "#000000"},
					BorderRight: cells.Border{Style: "thin", Color: "#000000"},
					BorderBottom: cells.Border{Style: "thin", Color: "#000000"},
					BorderLeft: cells.Border{Style: "thin", Color: "#000000"},
				},
			})
		}

		// --- Font Families (Rows 93-94) ---
		fontFamilies := []string{"Arial", "Helvetica", "Times New Roman", "Georgia", "Courier New", "Verdana"}
		for col, ff := range fontFamilies {
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 93, Col: col,
				Format: cells.Format{FontFamily: ff, FontSize: 12},
			})
			srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
				SheetID: sheet5.ID, Row: 94, Col: col,
				Format: cells.Format{FontFamily: ff, FontSize: 10, FontColor: "#666666"},
			})
		}

		// Set column widths for Formats sheet
		srv.SheetService().SetColWidth(ctx, sheet5.ID, 0, 180)
		srv.SheetService().SetColWidth(ctx, sheet5.ID, 1, 140)
		srv.SheetService().SetColWidth(ctx, sheet5.ID, 2, 140)
		srv.SheetService().SetColWidth(ctx, sheet5.ID, 3, 140)
		srv.SheetService().SetColWidth(ctx, sheet5.ID, 4, 120)
		srv.SheetService().SetColWidth(ctx, sheet5.ID, 5, 120)
	}

	// Create Charts demo sheet with all chart types
	if sheet6 != nil {
		// Data for various chart demonstrations
		// Section 1: Monthly Sales Data (A1:E7) - For Line, Area, Combo charts
		// Section 2: Regional Data (A9:C13) - For Column, Bar charts
		// Section 3: Category Distribution (A15:B19) - For Pie, Doughnut charts
		// Section 4: Team Performance (A21:F26) - For Radar charts
		// Section 5: XY Data (A28:C38) - For Scatter, Bubble charts
		// Section 6: Quarterly Breakdown (A40:D45) - For Stacked charts
		// Section 7: Stock Data (A47:E53) - For Candlestick charts
		// Section 8: Waterfall Data (A55:B61) - For Waterfall charts
		// Section 9: Distribution Data (A63:A83) - For Histogram charts
		// Section 10: Hierarchy Data (A85:C92) - For Treemap charts
		// Section 11: Gauge Data (A94:B97) - For Gauge charts

		chartsData := [][]interface{}{
			// Section 1: Monthly Sales Data (Rows 0-6)
			{"Month", "Sales", "Expenses", "Profit", "Growth %"},
			{"Jan", 45000, 32000, 13000, 0},
			{"Feb", 52000, 35000, 17000, 31},
			{"Mar", 48000, 33000, 15000, -12},
			{"Apr", 61000, 38000, 23000, 53},
			{"May", 55000, 36000, 19000, -17},
			{"Jun", 67000, 41000, 26000, 37},
			// Blank row (Row 7)
			{},
			// Section 2: Regional Data (Rows 8-12)
			{"Region", "Q1 Sales", "Q2 Sales"},
			{"North America", 125000, 142000},
			{"Europe", 98000, 115000},
			{"Asia Pacific", 87000, 105000},
			{"Latin America", 45000, 52000},
			// Blank row (Row 13)
			{},
			// Section 3: Category Distribution (Rows 14-18)
			{"Product Category", "Market Share %"},
			{"Electronics", 35},
			{"Software", 28},
			{"Services", 22},
			{"Hardware", 15},
			// Blank row (Row 19)
			{},
			// Section 4: Team Performance (Rows 20-25)
			{"Metric", "Team Alpha", "Team Beta", "Team Gamma", "Team Delta"},
			{"Speed", 85, 72, 90, 78},
			{"Quality", 92, 88, 75, 85},
			{"Efficiency", 78, 82, 88, 90},
			{"Innovation", 88, 75, 82, 70},
			{"Collaboration", 75, 90, 78, 88},
			// Blank row (Row 26)
			{},
			// Section 5: XY Scatter Data (Rows 27-37)
			{"Point", "X Value", "Y Value", "Size"},
			{"A", 10, 25, 15},
			{"B", 18, 38, 22},
			{"C", 25, 32, 18},
			{"D", 32, 48, 28},
			{"E", 40, 55, 35},
			{"F", 48, 62, 25},
			{"G", 55, 58, 30},
			{"H", 62, 75, 40},
			{"I", 70, 82, 32},
			{"J", 78, 88, 38},
			// Blank row (Row 38)
			{},
			// Section 6: Quarterly Breakdown (Rows 39-44)
			{"Quarter", "Hardware", "Software", "Services"},
			{"Q1 2024", 28000, 42000, 18000},
			{"Q2 2024", 32000, 48000, 22000},
			{"Q3 2024", 35000, 52000, 25000},
			{"Q4 2024", 42000, 58000, 30000},
			{"Q1 2025", 38000, 55000, 28000},
			// Blank row (Row 45)
			{},
			// Section 7: Stock/Candlestick Data (Rows 46-52)
			{"Date", "Open", "High", "Low", "Close"},
			{"Day 1", 100, 108, 98, 105},
			{"Day 2", 105, 112, 103, 110},
			{"Day 3", 110, 115, 105, 108},
			{"Day 4", 108, 118, 106, 116},
			{"Day 5", 116, 122, 112, 120},
			{"Day 6", 120, 125, 115, 118},
			// Blank row (Row 53)
			{},
			// Section 8: Waterfall Data (Rows 54-60)
			{"Category", "Amount"},
			{"Starting Balance", 50000},
			{"Product Sales", 35000},
			{"Service Revenue", 22000},
			{"Operating Costs", -28000},
			{"Marketing", -12000},
			{"Net Result", 67000},
			// Blank row (Row 61)
			{},
			// Section 9: Distribution Data (Rows 62-82) - For Histogram
			{"Test Scores"},
			{72}, {85}, {68}, {91}, {78}, {82}, {75}, {88}, {95}, {70},
			{83}, {77}, {89}, {64}, {92}, {80}, {86}, {73}, {79}, {87},
			// Blank row (Row 83)
			{},
			// Section 10: Hierarchy/Treemap Data (Rows 84-91)
			{"Category", "Subcategory", "Value"},
			{"Technology", "Software", 45000},
			{"Technology", "Hardware", 32000},
			{"Technology", "Services", 28000},
			{"Finance", "Banking", 38000},
			{"Finance", "Insurance", 25000},
			{"Retail", "Online", 42000},
			{"Retail", "Stores", 30000},
			// Blank row (Row 92)
			{},
			// Section 11: Gauge Data (Rows 93-96)
			{"Metric", "Value", "Target"},
			{"Customer Satisfaction", 78, 85},
			{"Revenue Growth", 92, 80},
			{"Market Share", 65, 75},
		}

		for row, rowData := range chartsData {
			for col, value := range rowData {
				if value == nil {
					continue
				}
				var formula string
				var cellValue interface{}

				if strVal, ok := value.(string); ok && len(strVal) > 0 && strVal[0] == '=' {
					formula = strVal
				} else {
					cellValue = value
				}

				srv.CellService().Set(ctx, sheet6.ID, row, col, &cells.SetCellIn{
					Value:   cellValue,
					Formula: formula,
				})
			}
		}

		// Apply section header formatting
		sectionHeaders := []int{0, 8, 14, 20, 27, 39, 46, 54, 62, 84, 93}
		for _, row := range sectionHeaders {
			colCount := 5
			if row == 14 || row == 54 || row == 62 {
				colCount = 2
			} else if row == 84 {
				colCount = 3
			}
			for col := 0; col < colCount; col++ {
				srv.CellService().SetFormat(ctx, &cells.SetFormatIn{
					SheetID: sheet6.ID,
					Row:     row,
					Col:     col,
					Format: cells.Format{
						Bold:            true,
						BackgroundColor: "#8B5CF6",
						FontColor:       "#FFFFFF",
						HAlign:          "center",
					},
				})
			}
		}

		// Set column widths
		srv.SheetService().SetColWidth(ctx, sheet6.ID, 0, 120)
		srv.SheetService().SetColWidth(ctx, sheet6.ID, 1, 100)
		srv.SheetService().SetColWidth(ctx, sheet6.ID, 2, 100)
		srv.SheetService().SetColWidth(ctx, sheet6.ID, 3, 100)
		srv.SheetService().SetColWidth(ctx, sheet6.ID, 4, 100)

		// Create comprehensive chart demonstrations
		chartsSvc := srv.ChartService()

		// 1. LINE CHART - Monthly Sales Trend (Google Sheets compatible)
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Monthly Sales Trend",
			ChartType: charts.ChartTypeLine,
			Position:  charts.Position{Row: 0, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 320},
			DataRanges: []charts.DataRange{{
				StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
				HasHeader: true,
			}},
			Title:    &charts.ChartTitle{Text: "Monthly Sales Trend", FontSize: 16, Bold: true},
			Subtitle: &charts.ChartTitle{Text: "Sales, Expenses & Profit", FontSize: 12},
			Legend:   &charts.LegendConfig{Enabled: true, Position: "bottom", Alignment: "center"},
			Axes: &charts.AxesConfig{
				XAxis: &charts.AxisConfig{
					Title:     &charts.ChartTitle{Text: "Month"},
					GridLines: false,
				},
				YAxis: &charts.AxisConfig{
					Title:     &charts.ChartTitle{Text: "Amount ($)"},
					GridLines: true,
				},
			},
			Series: []charts.SeriesConfig{
				{Name: "Sales", Color: "#4CAF50", PointStyle: "circle", PointRadius: 4, Tension: 0.3},
				{Name: "Expenses", Color: "#FF5722", PointStyle: "circle", PointRadius: 4, Tension: 0.3},
				{Name: "Profit", Color: "#2196F3", PointStyle: "circle", PointRadius: 4, Tension: 0.3},
			},
			Options: &charts.ChartOptions{
				Animated:          true,
				AnimationDuration: 750,
				TooltipEnabled:    true,
				Interactive:       true,
				BackgroundColor:   "#FFFFFF",
				HoverMode:         "index",
			},
		})

		// 2. COLUMN CHART - Regional Sales Comparison
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Regional Sales",
			ChartType: charts.ChartTypeColumn,
			Position:  charts.Position{Row: 0, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 480, Height: 320},
			DataRanges: []charts.DataRange{{
				StartRow: 8, StartCol: 0, EndRow: 12, EndCol: 2,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Regional Sales Comparison", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Axes: &charts.AxesConfig{
				XAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Region"}},
				YAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Sales ($)"}, GridLines: true},
			},
			Series: []charts.SeriesConfig{
				{Name: "Q1 Sales", Color: "#3F51B5"},
				{Name: "Q2 Sales", Color: "#00BCD4"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 3. BAR CHART - Horizontal Regional Comparison
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Regional Bar Chart",
			ChartType: charts.ChartTypeBar,
			Position:  charts.Position{Row: 18, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 480, Height: 300},
			DataRanges: []charts.DataRange{{
				StartRow: 8, StartCol: 0, EndRow: 12, EndCol: 2,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Regional Sales (Horizontal)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Series: []charts.SeriesConfig{
				{Name: "Q1 Sales", Color: "#673AB7"},
				{Name: "Q2 Sales", Color: "#E91E63"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 4. PIE CHART - Market Share Distribution
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Market Share Pie",
			ChartType: charts.ChartTypePie,
			Position:  charts.Position{Row: 18, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 420, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 14, StartCol: 0, EndRow: 18, EndCol: 1,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Product Market Share", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "right", Alignment: "center"},
			Series: []charts.SeriesConfig{
				{Name: "Electronics", Color: "#4CAF50"},
				{Name: "Software", Color: "#2196F3"},
				{Name: "Services", Color: "#FF9800"},
				{Name: "Hardware", Color: "#9C27B0"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 5. DOUGHNUT CHART - Market Share with center cutout
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Market Share Doughnut",
			ChartType: charts.ChartTypeDoughnut,
			Position:  charts.Position{Row: 36, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 420, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 14, StartCol: 0, EndRow: 18, EndCol: 1,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Market Share (Doughnut)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "right"},
			Options: &charts.ChartOptions{
				Animated:         true,
				TooltipEnabled:   true,
				Interactive:      true,
				CutoutPercentage: 50,
			},
		})

		// 6. AREA CHART - Sales Trend with fill
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Sales Area Chart",
			ChartType: charts.ChartTypeArea,
			Position:  charts.Position{Row: 36, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 320},
			DataRanges: []charts.DataRange{{
				StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Sales Trend (Area)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Series: []charts.SeriesConfig{
				{Name: "Sales", Color: "#4CAF50", Fill: true, Tension: 0.4},
				{Name: "Expenses", Color: "#FF5722", Fill: true, Tension: 0.4},
				{Name: "Profit", Color: "#2196F3", Fill: true, Tension: 0.4},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 7. SCATTER CHART - XY Correlation
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "XY Scatter Plot",
			ChartType: charts.ChartTypeScatter,
			Position:  charts.Position{Row: 54, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 480, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 27, StartCol: 1, EndRow: 37, EndCol: 2,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "X vs Y Correlation", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: false, Position: "none"},
			Axes: &charts.AxesConfig{
				XAxis: &charts.AxisConfig{
					Title:     &charts.ChartTitle{Text: "X Value"},
					GridLines: true,
				},
				YAxis: &charts.AxisConfig{
					Title:     &charts.ChartTitle{Text: "Y Value"},
					GridLines: true,
				},
			},
			Series: []charts.SeriesConfig{
				{Name: "Data Points", Color: "#3F51B5", PointStyle: "circle", PointRadius: 6},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 8. BUBBLE CHART - Multi-dimensional data
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Bubble Chart",
			ChartType: charts.ChartTypeBubble,
			Position:  charts.Position{Row: 54, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 480, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 27, StartCol: 1, EndRow: 37, EndCol: 3,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Bubble Chart (X, Y, Size)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: false, Position: "none"},
			Axes: &charts.AxesConfig{
				XAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "X Value"}, GridLines: true},
				YAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Y Value"}, GridLines: true},
			},
			Series: []charts.SeriesConfig{
				{Name: "Bubbles", Color: "#00BCD4"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 9. RADAR CHART - Team Performance
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Team Performance Radar",
			ChartType: charts.ChartTypeRadar,
			Position:  charts.Position{Row: 72, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 480, Height: 380},
			DataRanges: []charts.DataRange{{
				StartRow: 20, StartCol: 0, EndRow: 25, EndCol: 4,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Team Performance Comparison", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Series: []charts.SeriesConfig{
				{Name: "Team Alpha", Color: "#4CAF50", Fill: true},
				{Name: "Team Beta", Color: "#2196F3", Fill: true},
				{Name: "Team Gamma", Color: "#FF9800", Fill: true},
				{Name: "Team Delta", Color: "#E91E63", Fill: true},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 10. STACKED COLUMN CHART - Quarterly Revenue Breakdown
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Stacked Column Chart",
			ChartType: charts.ChartTypeStackedColumn,
			Position:  charts.Position{Row: 72, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 39, StartCol: 0, EndRow: 44, EndCol: 3,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Quarterly Revenue by Segment", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Series: []charts.SeriesConfig{
				{Name: "Hardware", Color: "#4CAF50", Stack: "stack1"},
				{Name: "Software", Color: "#2196F3", Stack: "stack1"},
				{Name: "Services", Color: "#FF9800", Stack: "stack1"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 11. STACKED BAR CHART - Horizontal Stacked
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Stacked Bar Chart",
			ChartType: charts.ChartTypeStackedBar,
			Position:  charts.Position{Row: 90, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 39, StartCol: 0, EndRow: 44, EndCol: 3,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Revenue by Segment (Horizontal)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Series: []charts.SeriesConfig{
				{Name: "Hardware", Color: "#673AB7", Stack: "stack1"},
				{Name: "Software", Color: "#00BCD4", Stack: "stack1"},
				{Name: "Services", Color: "#FFC107", Stack: "stack1"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 12. STACKED AREA CHART - Cumulative Trend
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Stacked Area Chart",
			ChartType: charts.ChartTypeStackedArea,
			Position:  charts.Position{Row: 90, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 3,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Sales Trend (Stacked Area)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Series: []charts.SeriesConfig{
				{Name: "Sales", Color: "#4CAF50", Fill: true, Stack: "stack1"},
				{Name: "Expenses", Color: "#FF5722", Fill: true, Stack: "stack1"},
				{Name: "Profit", Color: "#2196F3", Fill: true, Stack: "stack1"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 13. COMBO CHART - Mixed Line and Column
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Combo Chart",
			ChartType: charts.ChartTypeCombo,
			Position:  charts.Position{Row: 108, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 540, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 0, StartCol: 0, EndRow: 6, EndCol: 4,
				HasHeader: true,
			}},
			Title:    &charts.ChartTitle{Text: "Sales Performance (Combo)", FontSize: 16, Bold: true},
			Subtitle: &charts.ChartTitle{Text: "Bars: Sales/Expenses, Line: Growth %", FontSize: 11},
			Legend:   &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Axes: &charts.AxesConfig{
				YAxis:  &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Amount ($)"}, GridLines: true},
				Y2Axis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Growth %"}},
			},
			Series: []charts.SeriesConfig{
				{Name: "Sales", ChartType: charts.ChartTypeColumn, Color: "#4CAF50"},
				{Name: "Expenses", ChartType: charts.ChartTypeColumn, Color: "#FF5722"},
				{Name: "Profit", ChartType: charts.ChartTypeColumn, Color: "#2196F3"},
				{Name: "Growth %", ChartType: charts.ChartTypeLine, Color: "#9C27B0", BorderWidth: 3, PointRadius: 5},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 14. CANDLESTICK CHART - Stock Data (Google Sheets compatible)
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Candlestick Chart",
			ChartType: charts.ChartTypeCandlestick,
			Position:  charts.Position{Row: 108, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 500, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 46, StartCol: 0, EndRow: 52, EndCol: 4,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Stock Price Movement", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: false, Position: "none"},
			Axes: &charts.AxesConfig{
				XAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Trading Day"}},
				YAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Price ($)"}, GridLines: true},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 15. WATERFALL CHART - Financial Flow
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Waterfall Chart",
			ChartType: charts.ChartTypeWaterfall,
			Position:  charts.Position{Row: 126, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 54, StartCol: 0, EndRow: 60, EndCol: 1,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Financial Waterfall", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: false, Position: "none"},
			Axes: &charts.AxesConfig{
				YAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Amount ($)"}, GridLines: true},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 16. HISTOGRAM CHART - Distribution
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Histogram Chart",
			ChartType: charts.ChartTypeHistogram,
			Position:  charts.Position{Row: 126, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 480, Height: 350},
			DataRanges: []charts.DataRange{{
				StartRow: 62, StartCol: 0, EndRow: 82, EndCol: 0,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Test Score Distribution", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: false, Position: "none"},
			Axes: &charts.AxesConfig{
				XAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Score Range"}},
				YAxis: &charts.AxisConfig{Title: &charts.ChartTitle{Text: "Frequency"}, GridLines: true},
			},
			Series: []charts.SeriesConfig{
				{Name: "Frequency", Color: "#3F51B5"},
			},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 17. TREEMAP CHART - Hierarchical Data
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Treemap Chart",
			ChartType: charts.ChartTypeTreemap,
			Position:  charts.Position{Row: 144, Col: 6, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 520, Height: 380},
			DataRanges: []charts.DataRange{{
				StartRow: 84, StartCol: 0, EndRow: 91, EndCol: 2,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "Revenue by Category (Treemap)", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})

		// 18. GAUGE CHART - KPI Metrics
		chartsSvc.Create(ctx, &charts.CreateIn{
			SheetID:   sheet6.ID,
			Name:      "Gauge Chart",
			ChartType: charts.ChartTypeGauge,
			Position:  charts.Position{Row: 144, Col: 12, OffsetX: 0, OffsetY: 0},
			Size:      charts.Size{Width: 400, Height: 300},
			DataRanges: []charts.DataRange{{
				StartRow: 93, StartCol: 0, EndRow: 96, EndCol: 2,
				HasHeader: true,
			}},
			Title:  &charts.ChartTitle{Text: "KPI Dashboard", FontSize: 16, Bold: true},
			Legend: &charts.LegendConfig{Enabled: true, Position: "bottom"},
			Options: &charts.ChartOptions{Animated: true, TooltipEnabled: true, Interactive: true},
		})
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
		"Sheets", "Sales Data, Summary, Formulas Demo, Inventory, Formats, Charts",
	)
	Blank()
	Hint("Start the server with: spreadsheet serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && spreadsheet seed")
	Blank()

	return nil
}
