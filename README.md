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
| GET | `/api/users/{id}/stats` | Statistiques d'un utilisateur |
| GET | `/api/services` | Liste des services (filtres optionnels) |
| POST | `/api/services` | Créer une annonce (`X-User-ID` requis) |
| GET | `/api/services/{id}` | Détail d'un service |
| PUT | `/api/services/{id}` | Modifier son annonce (`X-User-ID` requis) |
| DELETE | `/api/services/{id}` | Supprimer son annonce (`X-User-ID` requis) |

### 1. Gestion des utilisateurs ✅

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

### 2. Gestion des annonces de services ✅

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

### 3. Système d'échange ✅

| Méthode | Path | Description |
|---------|------|-------------|
| POST | `/api/exchanges` | Créer une demande d'échange |
| GET | `/api/exchanges` | Liste des échanges |
| GET | `/api/exchanges/{id}` | Détail d'un échange |
| PUT | `/api/exchanges/{id}/accept` | Accepter une demande |
| PUT | `/api/exchanges/{id}/reject` | Refuser une demande |
| PUT | `/api/exchanges/{id}/complete` | Marquer comme terminé |
| PUT | `/api/exchanges/{id}/cancel` | Annuler |

**Règles :**
- Cycle de vie : `pending` → `accepted` → `completed` ; ou `rejected` / `cancelled`.
- Un utilisateur ne peut pas demander son propre service.
- Un service ne peut avoir qu'un seul échange `pending` ou `accepted` à la fois.
- Le demandeur doit avoir assez de crédits pour lancer la demande.
- À l'**acceptation** : crédits bloqués (débités du demandeur, pas encore crédités à l'offreur).
- À la **completion** : crédits transférés définitivement à l'offreur.
- À l'**annulation** d'un échange `accepted` : crédits restitués au demandeur.
- `accept` / `reject` : offreur uniquement ; `cancel` : demandeur ou offreur.
- `GET /api/exchanges?status={status}` : filtre côté serveur.
- `GET /api/exchanges` : nécessite `X-User-ID` (échanges envoyés + reçus).

### 4. Évaluations ✅

| Méthode | Path | Description |
|---------|------|-------------|
| POST | `/api/exchanges/{id}/review` | Donner un avis |
| GET | `/api/users/{id}/reviews` | Avis reçus par un utilisateur |
| GET | `/api/services/{id}/reviews` | Avis sur un service |

**Règles :**
- L'échange doit être au statut `completed`.
- Seuls le demandeur et l'offreur peuvent laisser un avis.
- Un seul avis par participant et par échange.
- Note entre `1` et `5` ; `commentaire` optionnel.
- `POST` nécessite `X-User-ID` (l'auteur évalue l'autre partie).

### 5. Tableau de bord / Statistiques ✅

| Méthode | Path | Description |
|---------|------|-------------|
| GET | `/api/users/{id}/stats` | Statistiques d'un utilisateur |

**Données retournées :**
- `services_actifs` : annonces actives de l'utilisateur
- `echanges_completes` : échanges terminés (demandeur ou offreur)
- `credit_balance` : solde actuel
- `note_moyenne` / `nb_avis` : avis reçus
- `total_gagne` : crédits gagnés via échanges (hors bienvenue)
- `total_depense` : crédits dépensés nets (`spend` − `refund`)

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

### Créer une demande d'échange (201)

```bash
curl -s -X POST http://localhost:8081/api/exchanges \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 2' \
  -d '{"service_id":1}'
```

### Accepter et terminer un échange

```bash
curl -s -X PUT http://localhost:8081/api/exchanges/1/accept -H 'X-User-ID: 1'
curl -s -X PUT http://localhost:8081/api/exchanges/1/complete -H 'X-User-ID: 1'
```

### Lister ses échanges

```bash
curl -s 'http://localhost:8081/api/exchanges' -H 'X-User-ID: 2'
curl -s 'http://localhost:8081/api/exchanges?status=pending' -H 'X-User-ID: 1'
```

### Laisser un avis sur un échange terminé (201)

```bash
curl -s -X POST http://localhost:8081/api/exchanges/1/review \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 2' \
  -d '{"note":5,"commentaire":"Excellent service"}'
```

### Lister les avis reçus / sur un service

```bash
curl -s http://localhost:8081/api/users/1/reviews
curl -s http://localhost:8081/api/services/1/reviews
```

### Statistiques d'un utilisateur

```bash
curl -s http://localhost:8081/api/users/1/stats
```

## Codes d'erreur

Toutes les erreurs JSON ont le même format :

```json
{ "error": "message explicite" }
```

### Mapping HTTP ↔ erreurs sentinelles

| Code HTTP | Sentinelle (`errors.go`) | Signification |
|-----------|--------------------------|---------------|
| `400 Bad Request` | `ErrValidation` | Données invalides (JSON, champs, règles métier) |
| `401 Unauthorized` | `ErrUnauthorized` | Header `X-User-ID` manquant ou invalide |
| `403 Forbidden` | `ErrForbidden` | Authentifié, mais pas autorisé sur cette ressource |
| `404 Not Found` | `ErrNotFound` | Ressource introuvable |
| `409 Conflict` | `ErrConflict` | Conflit d'état (unicité, réservation, avis doublon…) |
| `500 Internal Server Error` | *(autre)* | Erreur inattendue (détail masqué côté client) |

Les erreurs métier sont wrappées avec `%w` (ex. `données invalides: crédits insuffisants`) puis reconnues via `errors.Is` dans `writeServiceError`.

### Erreurs fréquentes par domaine

**Utilisateurs**

| Situation | HTTP | Message typique |
|-----------|------|-----------------|
| Pseudo vide | `400` | `données invalides: pseudo obligatoire` |
| Pseudo déjà pris | `409` | `conflit: pseudo déjà utilisé` |
| Utilisateur inexistant | `404` | `ressource introuvable` |
| Modifier le profil d'un autre | `403` | `accès interdit` |
| Niveau de skill invalide | `400` | `données invalides: niveau invalide (...)` |
| Corps JSON invalide | `400` | `corps JSON invalide` |
| `id` URL non numérique | `400` | `id invalide` |

**Services**

| Situation | HTTP | Message typique |
|-----------|------|-----------------|
| Sans `X-User-ID` | `401` | `header X-User-ID requis` |
| Catégorie hors liste | `400` | `données invalides: catégorie invalide` |
| Compétence non possédée | `400` | `données invalides: l'utilisateur n'a pas la compétence "..."` |
| Titre / durée / crédits invalides | `400` | `données invalides: ...` |
| Service inexistant | `404` | `ressource introuvable` |
| Modifier / supprimer l'annonce d'un autre | `403` | `accès interdit` |

**Échanges**

| Situation | HTTP | Message typique |
|-----------|------|-----------------|
| Sans `X-User-ID` | `401` | `header X-User-ID requis` |
| Demander son propre service | `400` | `données invalides: impossible de demander son propre service` |
| Crédits insuffisants | `400` | `données invalides: crédits insuffisants` |
| Service inactif | `400` | `données invalides: le service n'est pas actif` |
| Service déjà réservé (`pending`/`accepted`) | `409` | `conflit: le service est déjà réservé` |
| Accepter / refuser sans être l'offreur | `403` | `accès interdit` |
| Transition de statut invalide | `409` | `conflit: l'échange n'est pas en attente` (ex.) |
| Échange inexistant | `404` | `ressource introuvable` |
| Statut de filtre invalide | `400` | `données invalides: statut invalide` |

**Avis**

| Situation | HTTP | Message typique |
|-----------|------|-----------------|
| Sans `X-User-ID` | `401` | `header X-User-ID requis` |
| Échange non `completed` | `400` | `données invalides: l'échange doit être terminé pour laisser un avis` |
| Note hors 1–5 | `400` | `données invalides: la note doit être entre 1 et 5` |
| Auteur hors participants | `403` | `accès interdit` |
| 2ᵉ avis du même auteur | `409` | `conflit: avis déjà laissé pour cet échange` |

**Stats**

| Situation | HTTP | Message typique |
|-----------|------|-----------------|
| Utilisateur inexistant | `404` | `ressource introuvable` |
| `id` invalide | `400` | `id invalide` / `données invalides: id invalide` |

## Tests

Prérequis : PostgreSQL accessible (par ex. `docker compose up -d db`).

```bash
# démarrer uniquement la base
docker compose up -d db

# lancer les tests avec couverture
go test -v -cover ./...
```

Collections Postman :
- `postman/BarterSwap-Demo-Complete.postman_collection.json` — **parcours complet soutenance** (comme `demo.sh`)
- `postman/BarterSwap-Users.postman_collection.json`
- `postman/BarterSwap-Services.postman_collection.json`

Pour la démo : importer `BarterSwap-Demo-Complete`, lancer **Collection Runner** dans l'ordre. Les IDs (`aliceId`, `bobId`, `serviceId`…) sont sauvegardés automatiquement.

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

Cas couverts (échanges) :

| Cas | Résultat attendu |
|-----|------------------|
| Créer une demande valide | `201` + statut `pending` |
| Créer sans `X-User-ID` | `401` |
| Demander son propre service | `400` |
| Crédits insuffisants | `400` |
| Demander un échange sur un service déjà réservé | `409` |
| Accepter un échange | `200` + crédits bloqués |
| Terminer un échange accepté | `200` + crédits transférés à l'offreur |
| Annuler un échange accepté | `200` + crédits restitués |
| GET échange inexistant | `404` |
| Lister / filtrer par statut | filtrage serveur |

Cas couverts (évaluations) :

| Cas | Résultat attendu |
|-----|------------------|
| Laisser un avis sur un échange terminé | `201` |
| Laisser un avis sans `X-User-ID` | `401` |
| Laisser un avis sur un échange non terminé | `400` |
| Note invalide | `400` |
| Avis en double sur le même échange | `409` |
| Avis par un tiers | `403` |
| GET avis utilisateur / service inexistant | `404` |

Cas couverts (statistiques) :

| Cas | Résultat attendu |
|-----|------------------|
| GET stats utilisateur neuf | `200` + solde 10, compteurs à 0 |
| GET stats après échange terminé + avis | valeurs cohérentes (solde, gains, dépenses, note) |
| GET stats utilisateur inexistant | `404` |
| GET stats avec id invalide | `400` |

## Architecture

Organisation en un seul package avec séparation des responsabilités :

- **HTTP** : handlers, middlewares (`net/http`), sérialisation JSON
- **Métier** : règles de gestion, validations (`App`)
- **Stockage** : accès PostgreSQL via `database/sql`

Les handlers HTTP ne contiennent pas de logique métier.
