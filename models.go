package main

// User représente un compte BarterSwap.
type User struct {
	ID            int     `json:"id"`
	Pseudo        string  `json:"pseudo"`
	Bio           string  `json:"bio,omitempty"`
	Ville         string  `json:"ville,omitempty"`
	Skills        []Skill `json:"skills,omitempty"`
	CreditBalance int     `json:"credit_balance"`
	CreatedAt     string  `json:"created_at"`
}

// Skill représente une compétence proposée par un utilisateur.
type Skill struct {
	Nom    string `json:"nom"`    // ex: "Jardinage"
	Niveau string `json:"niveau"` // "débutant", "intermédiaire", "expert"
}

// Service représente une annonce de service proposée par un utilisateur.
type Service struct {
	ID           int    `json:"id"`
	ProviderID   int    `json:"provider_id"`
	Titre        string `json:"titre"`
	Description  string `json:"description,omitempty"`
	Categorie    string `json:"categorie"`
	DureeMinutes int    `json:"duree_minutes"` // durée estimée
	Credits      int    `json:"credits"`       // coût en crédits-temps
	Ville        string `json:"ville,omitempty"`
	Actif        bool   `json:"actif"`
	CreatedAt    string `json:"created_at"`
}

// Exchange représente une demande d'échange de service entre deux utilisateurs.
type Exchange struct {
	ID          int    `json:"id"`
	ServiceID   int    `json:"service_id"`
	RequesterID int    `json:"requester_id"`
	OwnerID     int    `json:"owner_id"`
	Status      string `json:"status"` // pending, accepted, rejected, cancelled, completed
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Review représente un avis laissé après un échange terminé.
type Review struct {
	ID          int    `json:"id"`
	ExchangeID  int    `json:"exchange_id"`
	AuthorID    int    `json:"author_id"`
	TargetID    int    `json:"target_id"`
	ServiceID   int    `json:"service_id"`
	Note        int    `json:"note"`
	Commentaire string `json:"commentaire,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// CreditTransaction trace une opération sur le solde de crédits-temps.
type CreditTransaction struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	ExchangeID int    `json:"exchange_id"`
	Montant    int    `json:"montant"` // positif = crédit, négatif = débit
	Type       string `json:"type"`    // "earn", "spend", "refund"
	CreatedAt  string `json:"created_at"`
}
