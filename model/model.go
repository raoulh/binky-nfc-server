package model

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raoulh/binky-server/config"
	"github.com/sirupsen/logrus"
)

type NFCCard struct {
	gorm.Model
	NFCID      string `gorm:"column:nfc_id"`
	PlaylistId int
}

var (
	db      *gorm.DB
	logging *logrus.Logger
)

func Init() error {
	logging = logrus.New()
	logging.Formatter = &logrus.TextFormatter{
		DisableTimestamp: true,
		QuoteEmptyFields: true,
	}
	logging.SetLevel(logrus.TraceLevel)
	log.SetOutput(os.Stdout)

	logging.Println("Using database:", config.Config.String("db.sqlite"))

	var err error
	db, err = gorm.Open(sqlite.Open(config.Config.String("db.sqlite")), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&NFCCard{})

	return nil
}

func GetAllPlaylistAssoc() (pls []NFCCard, err error) {
	result := db.Find(&pls)
	return pls, result.Error
}

func AddPlaylistAssoc(nfc string, playlist_id int) error {
	//first search if nfc tag already exists
	var c int64

	nfccard := NFCCard{
		NFCID:      nfc,
		PlaylistId: playlist_id,
	}

	result := db.Model(&nfccard).Where("nfc_id = ?", nfc).Count(&c)
	if result.Error != nil {
		return result.Error
	}
	if c > 0 {
		return fmt.Errorf("NFC id is already associated with a playlist, remove first")
	}

	result = db.Create(&nfccard)
	return result.Error
}

func DeletePlaylistAssoc(nfc string) error {
	nfccard := NFCCard{}
	result := db.Where("nfc_id = ?", nfc).First(&nfccard)
	if result.Error != nil {
		return result.Error
	}

	if nfccard.ID != 0 {
		return fmt.Errorf("nfc tag not found")
	}

	result = db.Delete(&nfccard)
	return result.Error
}

func GetPlaylistAssoc(nfc string) (*NFCCard, error) {
	nfccard := NFCCard{}
	result := db.Where("nfc_id = ?", nfc).First(&nfccard)
	if result.Error != nil {
		return nil, result.Error
	}

	return &nfccard, nil
}
