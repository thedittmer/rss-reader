package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"

	"github.com/thedittmer/rss-reader/internal/models"
)

type SheetsConfig struct {
	CredentialsFile string
	TokenFile       string
	SpreadsheetID   string
}

func NewSheetsConfig(dataDir string) *SheetsConfig {
	return &SheetsConfig{
		CredentialsFile: filepath.Join(dataDir, "credentials.json"),
		TokenFile:       filepath.Join(dataDir, "token.json"),
	}
}

func (s *Storage) getOrCreateRSSFolder(driveService *drive.Service) (string, error) {
	// Use the specific Sales shared drive folder ID
	companyFolderID := "17sE3dh1ujQtuecSLdutpTGsinpudD3Qb"

	// First, try to access the company folder to verify permissions
	_, err := driveService.Files.Get(companyFolderID).Fields("id").SupportsAllDrives(true).Do()
	if err != nil {
		// Clean up any potential trailing periods in the error message
		cleanFolderID := strings.TrimRight(companyFolderID, ".")
		return "", fmt.Errorf("service account cannot access the shared folder. Please follow these steps:\n"+
			"1. Open this folder: https://drive.google.com/drive/folders/%s\n"+
			"2. Click the 'Share' button\n"+
			"3. Add this email as a Content Manager: %s\n"+
			"4. Click 'Share'\n"+
			"5. Try exporting again\n"+
			"Error: %v", cleanFolderID, s.getServiceAccountEmail(), err)
	}

	// Search for RSS Reader folder within the shared folder
	query := fmt.Sprintf("name = 'RSS Reader' and mimeType = 'application/vnd.google-apps.folder' and trashed = false and '%s' in parents", companyFolderID)
	files, err := driveService.Files.List().Q(query).Spaces("drive").SupportsAllDrives(true).Do()
	if err != nil {
		return "", fmt.Errorf("unable to search for RSS Reader folder: %v", err)
	}

	// If RSS Reader folder exists, return its ID
	if len(files.Files) > 0 {
		return files.Files[0].Id, nil
	}

	// If we get here, we need to create the folder
	folder := &drive.File{
		Name:     "RSS Reader",
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{companyFolderID},
	}

	folder, err = driveService.Files.Create(folder).Fields("id").SupportsAllDrives(true).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create RSS Reader folder: %v", err)
	}

	return folder.Id, nil
}

// Add helper function to get service account email
func (s *Storage) getServiceAccountEmail() string {
	credentials, err := os.ReadFile(filepath.Join(s.dataDir, "credentials.json"))
	if err != nil {
		return "unknown"
	}

	var creds struct {
		ClientEmail string `json:"client_email"`
	}
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return "unknown"
	}

	return creds.ClientEmail
}

func (s *Storage) createNewSpreadsheet(sheetsService *sheets.Service, driveService *drive.Service, sheetsConfig *SheetsConfig) (string, error) {
	// Use the specific Sales shared drive folder ID
	companyFolderID := "17sE3dh1ujQtuecSLdutpTGsinpudD3Qb"

	// First, try to access the company folder to verify permissions
	_, err := driveService.Files.Get(companyFolderID).Fields("id").SupportsAllDrives(true).Do()
	if err != nil {
		// Clean up any potential trailing periods in the error message
		cleanFolderID := strings.TrimRight(companyFolderID, ".")
		return "", fmt.Errorf("service account cannot access the shared folder. Please follow these steps:\n"+
			"1. Open this folder: https://drive.google.com/drive/folders/%s\n"+
			"2. Click the 'Share' button\n"+
			"3. Add this email as a Content Manager: %s\n"+
			"4. Click 'Share'\n"+
			"5. Try exporting again\n"+
			"Error: %v", cleanFolderID, s.getServiceAccountEmail(), err)
	}

	// Generate a unique filename with timestamp
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	spreadsheetTitle := fmt.Sprintf("RSS Reader Articles - %s", timestamp)

	// Create new spreadsheet
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: spreadsheetTitle,
		},
		Sheets: []*sheets.Sheet{
			{
				Properties: &sheets.SheetProperties{
					Title: "Sheet1",
				},
			},
		},
	}

	spreadsheet, err = sheetsService.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create spreadsheet: %v", err)
	}

	// Move spreadsheet to the shared folder
	_, err = driveService.Files.Update(spreadsheet.SpreadsheetId, nil).AddParents(companyFolderID).Fields("id, parents").SupportsAllDrives(true).Do()
	if err != nil {
		return "", fmt.Errorf("unable to move spreadsheet to folder: %v", err)
	}

	// Get service account email from credentials
	credentials, err := os.ReadFile(sheetsConfig.CredentialsFile)
	if err != nil {
		return "", fmt.Errorf("unable to read credentials file: %v", err)
	}

	var creds struct {
		ClientEmail string `json:"client_email"`
	}
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return "", fmt.Errorf("unable to parse credentials: %v", err)
	}

	// Share with the service account
	permission := &drive.Permission{
		Type:         "user",
		Role:         "writer",
		EmailAddress: creds.ClientEmail,
	}

	// Try to share with the service account, but don't fail if it doesn't work
	// since the service account already has access through the shared drive
	_, err = driveService.Permissions.Create(spreadsheet.SpreadsheetId, permission).SupportsAllDrives(true).Do()
	if err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("Note: Could not explicitly share with service account (this is usually fine): %v\n", err)
	}

	return spreadsheet.SpreadsheetId, nil
}

// Add new type for export result
type ExportResult struct {
	SpreadsheetID string
	URL           string
	Error         error
}

func (s *Storage) ExportToSheets(articles []models.ArticleScore, spreadsheetID string) ExportResult {
	sheetsConfig := NewSheetsConfig(s.dataDir)

	// Load credentials
	credentials, err := os.ReadFile(sheetsConfig.CredentialsFile)
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to read credentials file: %v", err)}
	}

	// Configure the Google Sheets client with additional scopes
	oauthConfig, err := google.JWTConfigFromJSON(credentials,
		sheets.SpreadsheetsScope,
		drive.DriveScope,
		drive.DriveFileScope,
	)
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to parse credentials: %v", err)}
	}

	// Create clients
	client := oauthConfig.Client(context.Background())
	sheetsService, err := sheets.New(client)
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to create sheets client: %v", err)}
	}

	driveService, err := drive.New(client)
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to create drive client: %v", err)}
	}

	// If no spreadsheet ID provided, create a new one
	if spreadsheetID == "" {
		spreadsheetID, err = s.createNewSpreadsheet(sheetsService, driveService, sheetsConfig)
		if err != nil {
			return ExportResult{Error: err}
		}
		// Save the new spreadsheet ID
		if err := s.SaveSpreadsheetID(spreadsheetID); err != nil {
			return ExportResult{Error: fmt.Errorf("failed to save spreadsheet ID: %v", err)}
		}
	}

	// Prepare data for export
	var values [][]interface{}
	// Add header row
	values = append(values, []interface{}{
		"Title", "Link", "Source", "Published Date", "Score", "Exported Date",
	})

	// Add article data
	for _, article := range articles {
		// Format dates in a more readable way
		publishedDate := article.Item.Published.Format("2006-01-02 15:04:05")
		exportedDate := time.Now().Format("2006-01-02 15:04:05")

		values = append(values, []interface{}{
			article.Item.Title,
			article.Item.Link,
			article.Item.FeedSource,
			publishedDate,
			fmt.Sprintf("%.2f", article.Score),
			exportedDate,
		})
	}

	// Create the request
	range_ := "Sheet1!A1:F" + fmt.Sprintf("%d", len(values))
	valueRange := &sheets.ValueRange{
		Values: values,
	}

	// Update the spreadsheet
	_, err = sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		range_,
		valueRange,
	).ValueInputOption("RAW").Do()
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to update spreadsheet: %v", err)}
	}

	// Get the spreadsheet to find the sheet ID
	spreadsheet, err := sheetsService.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to get spreadsheet: %v", err)}
	}

	if len(spreadsheet.Sheets) == 0 {
		return ExportResult{Error: fmt.Errorf("spreadsheet has no sheets")}
	}

	sheetID := spreadsheet.Sheets[0].Properties.SheetId

	// Freeze the first row after the data is populated
	requests := []*sheets.Request{
		{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId: sheetID,
					GridProperties: &sheets.GridProperties{
						FrozenRowCount: 1,
					},
				},
				Fields: "gridProperties.frozenRowCount",
			},
		},
	}

	batchUpdate := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	_, err = sheetsService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdate).Do()
	if err != nil {
		return ExportResult{Error: fmt.Errorf("unable to freeze first row: %v", err)}
	}

	// Generate the spreadsheet URL
	spreadsheetURL := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit", spreadsheetID)

	return ExportResult{
		SpreadsheetID: spreadsheetID,
		URL:           spreadsheetURL,
		Error:         nil,
	}
}

func (s *Storage) SaveSpreadsheetID(id string) error {
	path := filepath.Join(s.dataDir, "spreadsheet.json")
	data := map[string]string{"id": id}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling spreadsheet ID: %v", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("error saving spreadsheet ID: %v", err)
	}

	return nil
}

func (s *Storage) LoadSpreadsheetID() (string, error) {
	path := filepath.Join(s.dataDir, "spreadsheet.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading spreadsheet ID: %v", err)
	}

	var config map[string]string
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("error parsing spreadsheet ID: %v", err)
	}

	return config["id"], nil
}
