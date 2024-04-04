package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/vynious/ascenda-lp-backend/db"
	"github.com/vynious/ascenda-lp-backend/types"
)

var (
	DB        *db.DBService
	batchsize = 100
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env")
	}

	var DB, err = db.SpawnDBService()
	if err != nil {
		log.Fatalf("Error spawn DB service...")
	}

	clearDatabase(DB)

	// TODO: Add all models to be migrated here
	models := []interface{}{&types.Transaction{}, &types.Points{}, &types.User{}, &types.Role{}, &types.RolePermission{}, types.ApprovalChainMap{}}
	if err := DB.Conn.AutoMigrate(models...); err != nil {
		log.Fatalf("Failed to auto-migrate models")
	}
	log.Print("Successfully auto-migrated models")

	defer DB.CloseConn()

	seedFile("users", DB)
	seedFile("points", DB)
	seedRolesAndPermissions(DB)
	seedApprovalChainMap(DB)
}

func seedFile(filename string, DB *db.DBService) {
	file, err := os.Open(fmt.Sprintf("./seed/data/%s.csv", filename))
	if err != nil {
		log.Fatalf("Error opening %s.csv: %v", filename, err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		log.Fatalf("Error reading %s.csv: %v", filename, err)
	}

	switch filename {
	case "points":
		seedPoints(records, DB)
	case "users":
		seedUsers(records, DB)
	default:
		log.Fatalf("Unsupported file type: %s", filename)
	}
}

func seedPoints(records [][]string, DB *db.DBService) {
	var pointsRecords []types.Points
	for i, record := range records {
		if i == 0 {
			continue
		}
		balance, _ := strconv.Atoi(record[2]) // convert to int
		data := types.Points{
			ID:      record[0],
			UserID:  record[1],
			Balance: int32(balance),
		}
		pointsRecords = append(pointsRecords, data)
	}

	res := DB.Conn.CreateInBatches(pointsRecords, batchsize)
	if res.Error != nil {
		log.Fatalf("Database error %s", res.Error)
	}
}

func seedUsers(records [][]string, DB *db.DBService) {
	var usersRecords []types.User
	for i, record := range records {
		if i == 0 {
			continue
		}
		data := types.User{
			Id:        record[0],
			Email:     record[1],
			FirstName: record[2],
			LastName:  record[3],
			// if no role specified, customer role (no admin access)
			// Role:      record[4],
		}
		usersRecords = append(usersRecords, data)
	}

	res := DB.Conn.CreateInBatches(usersRecords, batchsize)
	if res.Error != nil {
		log.Fatalf("Database error %s", res.Error)
	}
}

func clearDatabase(DB *db.DBService) {
	// Specify the order of deletion based on foreign key dependencies
	models := []interface{}{&types.RolePermission{}, &types.Transaction{}, &types.Points{}, types.ApprovalChainMap{}, &types.Role{}, &types.User{}}
	for _, model := range models {
		if result := DB.Conn.Unscoped().Where("1 = 1").Delete(model); result.Error != nil {
			log.Fatalf("Failed to clear table for model %v: %v", model, result.Error)
		}
	}
	log.Println("Successfully cleared the database")
}

func seedRolesAndPermissions(DB *db.DBService) {
	// Owner, Manager, Engineer, Product Manager
	var roles types.RoleList = types.RoleList{
		types.Role{
			RoleName: "owner",
			Permissions: types.RolePermissionList{
				types.RolePermission{
					Resource:  "user_storage",
					CanCreate: true,
					CanRead:   true,
					CanUpdate: true,
					CanDelete: true,
				},
				types.RolePermission{
					Resource:  "points_ledger",
					CanRead:   true,
					CanUpdate: true,
				},
				types.RolePermission{
					Resource: "logs",
					CanRead:  true,
				},
			},
		},
		types.Role{
			RoleName: "manager",
			Permissions: types.RolePermissionList{
				types.RolePermission{
					Resource:  "user_storage",
					CanCreate: true,
					CanRead:   true,
					CanUpdate: true,
				},
				types.RolePermission{
					Resource:  "points_ledger",
					CanRead:   true,
					CanUpdate: true,
				},
				types.RolePermission{
					Resource: "logs",
					CanRead:  true,
				},
			},
		},
		types.Role{
			RoleName: "engineer",
			Permissions: types.RolePermissionList{
				types.RolePermission{
					Resource: "user_storage",
					CanRead:  true,
				},
				types.RolePermission{
					Resource: "points_ledger",
					CanRead:  true,
				},
				types.RolePermission{
					Resource: "logs",
					CanRead:  true,
				},
			},
		},
		types.Role{
			RoleName: "product_manager",
			Permissions: types.RolePermissionList{
				types.RolePermission{
					Resource: "user_storage",
					CanRead:  true,
				},
				types.RolePermission{
					Resource: "points_ledger",
					CanRead:  true,
				},
			},
		},
	}
	for _, role := range roles {
		res := DB.Conn.Create(&role)
		if res.Error != nil {
			log.Fatalf("Error creating roles/permissions: %v", res.Error)
		}
	}
	log.Printf("Successful roles and perms seed")
}

func seedApprovalChainMap(DB *db.DBService) {
	var approvalChainMaps = []struct {
		MakerRoleName   string
		CheckerRoleName string
	}{
		{"product_manager", "owner"},
		{"engineer", "manager"},
		{"engineer", "owner"},
	}

	for _, acm := range approvalChainMaps {
		var makerRole, checkerRole types.Role

		// Find MakerRole and CheckerRole based on RoleName
		if err := DB.Conn.Where("role_name = ?", acm.MakerRoleName).First(&makerRole).Error; err != nil {
			log.Fatalf("Maker role not found: %s", acm.MakerRoleName)
		}

		if err := DB.Conn.Where("role_name = ?", acm.CheckerRoleName).First(&checkerRole).Error; err != nil {
			log.Fatalf("Checker role not found: %s", acm.CheckerRoleName)
		}

		newACM := types.ApprovalChainMap{
			MakerRoleID:   makerRole.Id,
			CheckerRoleID: checkerRole.Id,
		}

		// Create ApprovalChainMap entry
		res := DB.Conn.Create(&newACM)
		if res.Error != nil {
			log.Fatalf("Error creating approval chain map: %v", res.Error)
		}
	}
}
