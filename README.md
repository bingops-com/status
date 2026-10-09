# status

Page d'état publique de `lab.bingo` : pour chaque service exposé sur Internet,
elle dit s'il répond, sa disponibilité jour par jour et la liste des
interruptions relevées.

Un binaire Go sert l'API et l'interface React embarquée. Il ne sonde rien
lui-même : il lit les vérifications faites par [Gatus](https://gatus.io), en
garde un historique par jour et transforme les séries d'échecs en incidents.

## Ce que la page montre, et ce qu'elle ne montre pas

- Seuls les endpoints des groupes Gatus listés dans `groups` sont lus,
  enregistrés et affichés. Un service d'un autre groupe n'apparaît nulle part.
- D'une vérification, seuls le nom du service, le résultat et la durée sont
  conservés. L'adresse sondée, les conditions et les messages d'erreur de
  Gatus ne sont ni stockés ni envoyés au navigateur ; l'adresse de Gatus non
  plus.
- La page est en lecture seule : pas de compte, pas de formulaire, aucune
  ressource chargée depuis un autre site.

## Fonctionnement

À chaque intervalle (`interval`, 60 s par défaut) le serveur lit
`<gatus>/api/v1/endpoints/statuses`. Chaque vérification n'est comptée qu'une
fois, même si deux lectures la contiennent.

- **Disponibilité** : part des vérifications réussies, par jour (dans le fuseau
  `timezone`) et sur l'ensemble des jours conservés (`days`, 90 par défaut).
- **Incident** : ouvert après `failureThreshold` échecs consécutifs (2 par
  défaut, pour qu'une réponse lente isolée ne compte pas), daté du premier
  échec de la série, fermé par la première réussite.
- **Mesures périmées** : si la dernière lecture réussie de Gatus date de plus
  de trois intervalles, l'état de chaque service devient « inconnu » et la page
  le dit, plutôt que d'afficher un dernier état peut-être faux.

L'historique est écrit dans `$STATUS_DATA/history.json` après chaque lecture.
Le perdre remet les chiffres à zéro ; rien d'autre n'en dépend. La page ne
remonte pas plus loin que son propre historique : au premier démarrage, les
jours passés sont « sans mesure ».

## Configuration

`config/status.yaml` :

```yaml
title: lab.bingo
gatus: http://gatus.gatus.svc.cluster.local:8080
groups: [Public]          # groupes Gatus rendus publics
interval: 60s
days: 90
timezone: Europe/Paris
failureThreshold: 2
services:                 # présentation ; name est le nom de l'endpoint Gatus
  - name: RomM
    description: Bibliothèque de jeux
    url: https://rom.lab.bingo
notices:                  # annonces rédigées à la main, la plus récente d'abord
  - date: 2026-10-12
    title: Maintenance du stockage
    body: Les services seront indisponibles une dizaine de minutes vers 22 h.
```

Un service présent dans Gatus mais absent de `services` est affiché avec son
seul nom. Pour ajouter un service à la page, ajoutez son endpoint au groupe
public de Gatus.

| Variable | Rôle | Défaut |
| --- | --- | --- |
| `STATUS_CONFIG` | Fichier de configuration | `config/status.yaml` |
| `STATUS_DATA` | Dossier de l'historique | `data` |
| `STATUS_ADDR` | Adresse d'écoute | `:3000` |
| `STATUS_GATUS` | Remplace `gatus` de la configuration | |
| `STATUS_DEMO` | `true` : historique inventé, sans Gatus ; `outage` : idem avec un service en panne | |

Le mode démonstration n'écrit jamais dans `STATUS_DATA` et la page l'annonce.

## API

- `GET /api/status` : état global, services (état, disponibilité, historique
  par jour), incidents et annonces. Mis en cache 15 s.
- `GET /healthz` : `ok`.

## Développement

```sh
make demo    # http://localhost:3000 avec un historique inventé
make dev     # idem, plus Vite avec rechargement à chaud sur :5173
make check   # tests Go, vet, vérification des types
```

## Apparence

La page suit la direction artistique de lab.bingo, sobre et moderne dans un
univers maritime teinté de piraterie
(`.agents/skills/labops-art-direction/SKILL.md`). Un bandeau de mer porte
l'état général en une phrase ; sa houle est l'état du lab, presque plate quand
tout répond, plus creusée quand des services ne répondent plus. Dessous, un
relevé sans cartes : une ligne par service avec une marque par jour, pleine
hauteur en bleu pour un jour sans interruption, plus courte et colorée sinon,
puis le journal de bord des interruptions. Le chat du lab, cache-œil compris,
regarde par un hublot dans le pied de page. Le texte existe en français et en
anglais, selon la langue du navigateur ; tout se fige si le système demande de
réduire les animations.

## Ce qui reste hors de ce dépôt

Le déploiement suit le même schéma que le portail et appartient au dépôt
`labops` : image (`docker/status`), manifestes (`apps/workloads/status`),
volume pour `STATUS_DATA`, nom DNS et route du tunnel Cloudflare. Rien de cela
n'existe encore.
