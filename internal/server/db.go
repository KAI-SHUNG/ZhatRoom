package server

import (
	"ZhatRoom/internal/protocol"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type storage struct {
	DB *gorm.DB
}

func InitDB() *storage {
	dsn := "host=127.0.0.1 user=postgres dbname=zhat_db password=zhatroom port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	db.AutoMigrate(&protocol.User{}, &protocol.Message{})

	return &storage{DB: db}
}

func (s *storage) NewMessage(msg protocol.Message) error {
	return s.DB.Create(&msg).Error
}

func (s *storage) UserExists(id string) (bool, error) {
	var count int64
	err := s.DB.Model(&protocol.User{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *storage) NewUser(id string, nickname string) error {
	user := protocol.User{
		ID:       id,
		Nickname: nickname,
	}
	return s.DB.Create(&user).Error
}

func (s *storage) Close() error {
	db, err := s.DB.DB()
	if db == nil {
		return nil
	}
	if err != nil {
		return err
	}
	return db.Close()
}
