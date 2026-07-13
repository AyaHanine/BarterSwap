# Tâches à faire — BarterSwap

Fonctionnalités déjà livrées : utilisateurs, annonces de services, système d'échange.

## 4. Évaluations

- [ ] Ajouter la table `reviews` (migration dans `db.go`)
  - `id`, `exchange_id`, `author_id`, `target_id`, `service_id`, `note` (1–5), `commentaire`, `created_at`
  - Contrainte : un seul avis par utilisateur et par échange
- [ ] Ajouter le modèle `Review` dans `models.go`
- [ ] Implémenter la couche store (`store_review.go`)
- [ ] Implémenter la logique métier (`service_review.go`)
  - L'échange doit être au statut `completed`
  - Seuls le demandeur et l'offreur peuvent laisser un avis
  - L'auteur ne peut pas s'évaluer lui-même
- [ ] Ajouter les handlers HTTP (`handlers_review.go`)
  - `POST /api/exchanges/{id}/review` — donner un avis (`X-User-ID` requis)
  - `GET /api/users/{id}/reviews` — avis reçus par un utilisateur
  - `GET /api/services/{id}/reviews` — avis sur un service
- [ ] Enregistrer les routes dans `router.go`
- [ ] Écrire les tests (`reviews_test.go`)
- [ ] Mettre à jour le README (section 4 avec ✅)
- [ ] Ajouter une collection Postman

## 5. Tableau de bord / Statistiques

- [ ] Implémenter `GET /api/users/{id}/stats`
  - Nombre d'échanges complétés (envoyés / reçus)
  - Solde de crédits actuel
  - Note moyenne reçue (dépend de la section 4)
  - Nombre d'annonces actives
- [ ] Ajouter handler, service et store dédiés
- [ ] Écrire les tests (`stats_test.go`)
- [ ] Mettre à jour le README (section 5 avec ✅)

## Améliorations optionnelles

- [ ] Collection Postman pour les échanges
- [ ] Documentation des codes d'erreur dans le README
- [ ] Pagination sur `GET /api/services` et `GET /api/exchanges`
