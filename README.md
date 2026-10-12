# status

Page d'état publique de `lab.bingo` : pour chaque service exposé sur Internet,
elle dit s'il répond, sa disponibilité heure par heure sur la semaine et jour
par jour au-delà, et la liste des interruptions relevées.

Un binaire Go sert l'API et l'interface React embarquée. Il ne sonde rien
lui-même : il lit les vérifications faites par [Gatus](https://gatus.io), en
garde un historique par jour, et par heure sur la dernière semaine, et transforme les séries d'échecs en incidents.

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
  Les 7 derniers jours sont aussi comptés par heure. Un historique écrit avant
  ce décompte n'a pas d'heures : la vue par heure se remplit à partir du
  déploiement.
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

### Reprise après un historique illisible

L'historique est un état généré localement, non sauvegardé, sans identifiant
secret. Un fichier JSON vide, tronqué ou invalide ne bloque plus la page : au
démarrage, il est déplacé sur le même volume vers
`$STATUS_DATA/history.json.corrupt-<suffixe unique>`, sans écraser une archive
existante. Un avertissement indique uniquement le chemin conservé. La page
charge la copie de secours `history.json.bak` et répare le fichier principal
avant de servir les mesures. Si aucun fichier valide ne subsiste, la page
redémarre sans mesures passées, avec un état « inconnu » jusqu'à la prochaine
lecture réussie de Gatus. Une copie de secours invalide est elle aussi
conservée sous un nom unique. Aucun chiffre historique n'est inventé.

Le fichier défectueux est conservé tel quel pour permettre à l'opérateur LabOps
d'en tenter la récupération hors ligne ; il n'est ni envoyé aux visiteurs ni
commité. Une nouvelle relance sans `history.json` est sûre, même avant la
première lecture. Un historique valide, y compris ancien sans compteurs
horaires, reste chargé normalement. Les erreurs de lecture, de permissions,
de stockage ou d'archivage restent bloquantes : elles ne déclenchent pas une
remise à zéro silencieuse.

Les nouvelles sauvegardes utilisent un fichier temporaire unique, synchronisé
sur disque avant remplacement atomique, puis une synchronisation du répertoire.
Une copie `history.json.bak` est ensuite actualisée de la même manière.
Les sauvegardes concurrentes d'une même instance sont sérialisées. Conserver
une seule instance qui écrit sur le volume (`Recreate` dans LabOps).

La récupération appartient à l'image du serveur, pas à une manipulation du
PVC : fusionner la PR source construit une image et ouvre la PR de déploiement
LabOps ; seule la fusion de cette dernière déploie le correctif. Ne pas supprimer
le PVC ni effacer `history.json` pour débloquer le démarrage.

Après déploiement, ces contrôles publics en lecture seule ne révèlent aucun
contenu du volume :

```sh
curl -fsS https://status.lab.bingo/healthz
curl -fsS https://status.lab.bingo/api/status
```

Vérifier que la page est disponible et que les mesures reprennent après un
intervalle de lecture. Les archives restent sur le PVC et ne sont pas
sauvegardées hors du volume : perdre ce volume perd aussi ces sources de
récupération et la copie `history.json.bak`. L'opérateur surveille leur taille ;
aucune archive de corruption n'est supprimée automatiquement. Le système de
fichiers doit prendre en charge la synchronisation des répertoires Linux.

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

- `GET /api/status` : état global, services (état, disponibilité sur 1, 7, 30
  jours et sur tout l'historique, historique par jour, et par heure sur la
  semaine), incidents et annonces. Mis en cache 15 s.
- `GET /healthz` : `ok`.

## Aperçu de lien

Un lien vers la page collé dans Discord (ou tout service qui lit les balises
Open Graph) affiche l'état du moment : le serveur écrit dans l'en-tête HTML un
titre (`🟢 Tous les services sont opérationnels`, `🟠 1 service perturbé`,
`🔴 Tous les services sont en panne`, `⚪ État inconnu`), puis une ligne par
service et une légende :

```
🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩🟧  RomM · 99,73 % · ne répond pas
7 derniers jours · incident en cours depuis 12:17
```

Les carrés reprennent les marques de la page (vert sans interruption, orange
moins de 30 minutes, rouge au-delà, blanc sans mesure), regroupées pour tenir
sur une ligne : 12 carrés de 2 h pour la journée, 14 de 12 h pour la semaine,
15 pour 30 jours ou tout l'historique. La période est celle du lien :
`?range=day`, `week` (par défaut), `month` ou `all`, que la page écrit dans
son adresse quand on change de période. Pour le robot
de Discord, `theme-color` prend la couleur de l'état, qui devient celle de la
barre de l'aperçu ; les navigateurs gardent le bleu de la page.

L'aperçu est en français et sans image. Discord le garde en cache : un lien
déjà collé ne se met pas à jour, et un nouveau collage peut montrer un état
vieux de quelques minutes à quelques heures.

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
relevé sans cartes : une ligne par service avec une marque par heure ou par
jour selon la période choisie (24 heures, 7 jours, 30 jours ou tout
l'historique), pleine hauteur en bleu sans interruption, plus courte et
colorée sinon, puis le journal de bord des interruptions. Le chat du lab, cache-œil compris,
regarde par un hublot dans le pied de page. Le texte existe en français et en
anglais, selon la langue du navigateur ; tout se fige si le système demande de
réduire les animations.

## Déploiement

Le déploiement appartient au dépôt `labops` et suit le schéma du portail :
image (`docker/status`), chart Helm (`charts/status`) et ses valeurs
(`apps/workloads/status`), nom DNS et route
du tunnel Cloudflare. Une modification du code poussée sur `master` demande à
`labops` de reconstruire l'image (`.github/workflows/trigger-rebuild.yml`) ;
la publication ouvre là-bas une pull request de déploiement.

Ce déclenchement a besoin du secret `REPO_INFRA_TOKEN` de l'organisation
`bingops-com`, rendu lisible par ce dépôt : un jeton
limité à `bingops-com/labops` avec `Contents: read and write`, comme pour le
portail. Sa création et sa rotation sont décrites pas à pas dans
`docker/status/README.md` de `labops`, avec les autres prérequis.
