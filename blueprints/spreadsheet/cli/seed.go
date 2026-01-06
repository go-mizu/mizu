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

	srv.SheetService().Create(ctx, &sheets.CreateIn{
		WorkbookID: wb.ID,
		Name:       "Charts",
		Index:      2,
		Color:      "#3B82F6",
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
		"Sheets", "Sales Data, Summary, Charts",
	)
	Blank()
	Hint("Start the server with: spreadsheet serve")
	Hint("Login with: alice@example.com / password123")
	Hint("To reset: rm -rf " + dataDir + " && spreadsheet seed")
	Blank()

	return nil
}
