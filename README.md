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

L'API est disponible sur `http://localhost:8081` (PostgreSQL exposé sur le port `5435` pour éviter les conflits locaux).

Vérifier que le service répond :

```bash
curl http://localhost:8081/health
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
| POST | `/api/users` | Créer un compte (10 crédits de bienvenue) |
| GET | `/api/users/{id}` | Profil public d'un utilisateur |
| PUT | `/api/users/{id}` | Modifier son profil (`X-User-ID` requis) |
| GET | `/api/users/{id}/skills` | Compétences d'un utilisateur |
| PUT | `/api/users/{id}/skills` | Définir ses compétences (`X-User-ID` requis) |
| GET | `/api/services` | Liste des services (filtres optionnels) |
| POST | `/api/services` | Créer une annonce (`X-User-ID` requis) |
| GET | `/api/services/{id}` | Détail d'un service |
| PUT | `/api/services/{id}` | Modifier son annonce (`X-User-ID` requis) |
| DELETE | `/api/services/{id}` | Supprimer son annonce (`X-User-ID` requis) |

### 1. Gestion des utilisateurs

| Méthode | Path | Description |
|---------|------|-------------|
| POST | `/api/users` | Créer un compte (crédits de bienvenue attribués automatiquement) |
| GET | `/api/users/{id}` | Profil public d'un utilisateur |
| PUT | `/api/users/{id}` | Modifier son profil |
| GET | `/api/users/{id}/skills` | Compétences d'un utilisateur |
| PUT | `/api/users/{id}/skills` | Définir ses compétences |

**Règles :**
- À la création, **10 crédits de bienvenue** sont attribués (journalisés dans `credit_transactions`).
- Les skills sont **écrasées** à chaque `PUT` (pas d'ajout individuel).
- Niveaux acceptés : `débutant`, `intermédiaire`, `expert`.
- `PUT` nécessite le header `X-User-ID` égal à `{id}`.

### 2. Gestion des annonces de services

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/api/services` | Liste des services (avec filtres optionnels) |
| POST | `/api/services` | Créer une annonce de service |
| GET | `/api/services/{id}` | Détail d'un service |
| PUT | `/api/services/{id}` | Modifier son annonce |
| DELETE | `/api/services/{id}` | Supprimer son annonce |
| GET | `/api/services?categorie={cat}` | Filtrer par catégorie |
| GET | `/api/services?ville={ville}` | Filtrer par ville |
| GET | `/api/services?search={mot-clé}` | Recherche textuelle |

**Règles :**
- Un utilisateur publie une annonce liée à **une de ses compétences** (la `categorie` doit correspondre à un skill).
- Publier avec une compétence que l'utilisateur n'a pas → `400`.
- Catégories fermées : `Informatique`, `Jardinage`, `Bricolage`, `Cuisine`, `Musique`, `Langues`, `Sport`, `Tutorat`, `Déménagement`, `Photographie`, `Animalier`, `Couture`, `Autre`.
- Filtrage / recherche **côté serveur** via query params.
- `POST` / `PUT` / `DELETE` nécessitent `X-User-ID` = propriétaire de l'annonce.

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

### Créer un utilisateur (201)

```bash
curl -s -X POST http://localhost:8081/api/users \
  -H 'Content-Type: application/json' \
  -d '{"pseudo":"alice","bio":"Jardinage & bricolage","ville":"Lyon"}'
```

### Définir ses compétences

```bash
curl -s -X PUT http://localhost:8081/api/users/1/skills \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"skills":[{"nom":"Jardinage","niveau":"expert"},{"nom":"Cuisine","niveau":"débutant"}]}'
```

### Créer une annonce de service (201)

```bash
curl -s -X POST http://localhost:8081/api/services \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"titre":"Taille de haies","description":"Je taille vos haies","categorie":"Jardinage","duree_minutes":60,"credits":2,"ville":"Lyon"}'
```

### Publier sans la compétence → 400

```bash
curl -s -X POST http://localhost:8081/api/services \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"titre":"Cours de piano","categorie":"Musique","duree_minutes":45,"credits":3}'
```

### Lister / filtrer les services

```bash
curl -s 'http://localhost:8081/api/services'
curl -s 'http://localhost:8081/api/services?categorie=Jardinage'
curl -s 'http://localhost:8081/api/services?ville=Lyon'
curl -s 'http://localhost:8081/api/services?search=haies'
```

### Modifier / supprimer son annonce

```bash
curl -s -X PUT http://localhost:8081/api/services/1 \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 1' \
  -d '{"titre":"Taille de haies (pro)","credits":3}'

curl -s -o /dev/null -w '%{http_code}\n' -X DELETE http://localhost:8081/api/services/1 \
  -H 'X-User-ID: 1'
```

## Tests

Prérequis : PostgreSQL accessible (par ex. `docker compose up -d db`).

```bash
# démarrer uniquement la base
docker compose up -d db

# lancer les tests avec couverture
go test -v -cover ./...
```

Collections Postman :
- `postman/BarterSwap-Users.postman_collection.json`
- `postman/BarterSwap-Services.postman_collection.json`

Cas couverts (utilisateurs) :

| Cas | Résultat attendu |
|-----|------------------|
| Créer un utilisateur | `201` + `credit_balance = 10` |
| Créer un utilisateur avec pseudo vide | `400` |
| Pseudo déjà utilisé | `409` |
| GET profil inexistant | `404` |
| PUT sans `X-User-ID` | `401` |
| PUT avec un autre `X-User-ID` | `403` |
| PUT skills (écrasement) | `200` + liste remplacée |
| Niveau de skill invalide | `400` |

Cas couverts (services) :

| Cas | Résultat attendu |
|-----|------------------|
| Créer un service avec une compétence possédée | `201` |
| Créer un service sans la compétence | `400` |
| Créer sans `X-User-ID` | `401` |
| Catégorie / titre / durée / crédits invalides | `400` |
| GET service inexistant | `404` |
| PUT / DELETE d'un autre utilisateur | `403` |
| Filtres `categorie`, `ville`, `search` | filtrage serveur |

## Architecture

Organisation en un seul package avec séparation des responsabilités :

- **HTTP** : handlers, middlewares (`net/http`), sérialisation JSON
- **Métier** : règles de gestion, validations (`App`)
- **Stockage** : accès PostgreSQL via `database/sql`

Les handlers HTTP ne contiennent pas de logique métier.
