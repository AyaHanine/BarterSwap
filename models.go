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

// CreditTransaction trace une opération sur le solde de crédits-temps.
type CreditTransaction struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	ExchangeID int    `json:"exchange_id"`
	Montant    int    `json:"montant"` // positif = crédit, négatif = débit
	Type       string `json:"type"`    // "earn", "spend", "refund"
	CreatedAt  string `json:"created_at"`
}
