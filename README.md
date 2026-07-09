# BarterSwap — API d'échange de compétences

BarterSwap est une plateforme qui permet à des particuliers d'échanger leurs compétences sans transaction monétaire. Le système fonctionne avec un **crédit-temps** : chaque heure de service rendue donne droit à une heure de service reçue.

Ce n'est pas :
- Une plateforme de freelance (pas d'argent)
- Une API de tutorat (pas limité à l'éducation)
- Un réseau social (pas de fil d'actualité, pas de likes)
- Un système de troc direct (les échanges sont différés via les crédits)

C'est une **banque de temps** : le temps est la monnaie d'échange.

## Contraintes techniques

- **Langage** : Go uniquement
- **Base de données** : PostgreSQL
- **Driver** : `github.com/lib/pq` (seule dépendance externe autorisée)
- **Pas d'ORM** : `database/sql` uniquement
- **Structure** : un seul package Go
- **Pas de mutex** : la base de données gère la concurrence
- **Pas de framework** : stdlib `net/http` uniquement
- **Auth** : header `X-User-ID`

## Installation

### Avec Docker (recommandé)

```bash
git clone <url>
cd BarterSwap
docker compose up --build
```

L'API est disponible sur `http://localhost:8080`.

Vérifier que le service répond :

```bash
curl http://localhost:8080/health
```

### Sans Docker

Prérequis : Go 1.22+, PostgreSQL.

```bash
git clone <url>
cd BarterSwap
cp .env.example .env
# démarrer PostgreSQL et adapter DATABASE_URL si besoin
go mod tidy
go run .
```

## Endpoints

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/health` | Santé du service |

Les endpoints métier (utilisateurs, services, échanges, avis, stats) seront documentés au fur et à mesure de leur implémentation.

### 1. Gestion des utilisateurs (à venir)

| Méthode | Path | Description |
|---------|------|-------------|
| POST | `/api/users` | Créer un compte (10 crédits de bienvenue) |
| GET | `/api/users/{id}` | Profil public d'un utilisateur |
| PUT | `/api/users/{id}` | Modifier son profil |
| GET | `/api/users/{id}/skills` | Compétences d'un utilisateur |
| PUT | `/api/users/{id}/skills` | Définir ses compétences |

### 2. Gestion des annonces de services (à venir)

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/api/services` | Liste des services (filtres optionnels) |
| POST | `/api/services` | Créer une annonce de service |
| GET | `/api/services/{id}` | Détail d'un service |
| PUT | `/api/services/{id}` | Modifier son annonce |
| DELETE | `/api/services/{id}` | Supprimer son annonce |

### 3. Système d'échange (à venir)

| Méthode | Path | Description |
|---------|------|-------------|
| POST | `/api/exchanges` | Créer une demande d'échange |
| GET | `/api/exchanges` | Liste des échanges |
| GET | `/api/exchanges/{id}` | Détail d'un échange |
| PUT | `/api/exchanges/{id}/accept` | Accepter une demande |
| PUT | `/api/exchanges/{id}/reject` | Refuser une demande |
| PUT | `/api/exchanges/{id}/complete` | Marquer comme terminé |
| PUT | `/api/exchanges/{id}/cancel` | Annuler |

### 4. Évaluations (à venir)

| Méthode | Path | Description |
|---------|------|-------------|
| POST | `/api/exchanges/{id}/review` | Donner un avis |
| GET | `/api/users/{id}/reviews` | Avis reçus par un utilisateur |
| GET | `/api/services/{id}/reviews` | Avis sur un service |

### 5. Tableau de bord / Statistiques (à venir)

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/api/users/{id}/stats` | Statistiques d'un utilisateur |

## Exemples d'utilisation

```bash
# Santé du service
curl -s http://localhost:8080/health
```

## Tests

```bash
go test -v -cover ./...
```

## Architecture

Organisation en un seul package avec séparation des responsabilités :

- **HTTP** : handlers, middlewares (`net/http`), sérialisation JSON
- **Métier** : règles de gestion, validations (service)
- **Stockage** : accès PostgreSQL via `database/sql`

Les handlers HTTP ne contiennent pas de logique métier.
