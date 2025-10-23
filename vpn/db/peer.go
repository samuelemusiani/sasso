package db

type Peer struct {
	ID         uint   `gorm:"primaryKey"`
	PrivateKey string `gorm:"not null"`
	PublicKey  string `gorm:"not null"`
	Address    string `gorm:"not null;unique"`
	UserID     uint   `gorm:"index"`

	Subnet []Subnet `gorm:"many2many:subnet_peers;constraint:OnDelete:CASCADE;"`
}

func initPeers() error {
	return db.AutoMigrate(&Peer{})
}

func NewPeer(privateKey, publicKey, address string, userID uint) error {
	iface := &Peer{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
		UserID:     userID,
	}
	if err := db.Create(iface).Error; err != nil {
		return err
	}
	return nil
}

func GetPeerByID(id uint) (*Peer, error) {
	var iface Peer
	if err := db.First(&iface, id).Error; err != nil {
		return nil, err
	}
	return &iface, nil
}

func GetPeersByUserID(userID uint) ([]Peer, error) {
	var ifaces []Peer
	if err := db.Where("user_id = ?", userID).Find(&ifaces).Error; err != nil {
		return nil, err
	}
	return ifaces, nil
}

func GetPeerByAddress(address string) (*Peer, error) {
	var iface Peer
	if err := db.First(&iface, "address = ?", address).Error; err != nil {
		return nil, err
	}
	return &iface, nil
}

func GetAllAddresses() ([]string, error) {
	var addresses []string
	if err := db.Model(&Peer{}).Pluck("address", &addresses).Error; err != nil {
		return nil, err
	}
	return addresses, nil
}

func GetAllPeers() ([]Peer, error) {
	var ifaces []Peer
	if err := db.Find(&ifaces).Error; err != nil {
		return nil, err
	}
	return ifaces, nil
}
