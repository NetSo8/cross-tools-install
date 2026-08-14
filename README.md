# Cross Tools Install

Installateur cross-platform en Go pour les outils de reverse engineering et de développement bas niveau. L'OS est détecté automatiquement et les paquets sont choisis depuis `tools.json`.

## Prérequis

- Go 1.26 ou plus récent pour compiler le projet
- Un gestionnaire de paquets présent, ou installable automatiquement, sur la machine :
  - Windows : `scoop` ou `winget`
  - Linux : `apt`, `dnf`, `pacman` ou `snap`
  - macOS : `brew`
- `pip`/`pip3` si les outils Python sont sélectionnés
- Les droits administrateur pour les installations système Linux

macOS fournit déjà LLDB, `otool` et DTrace via les outils système. Ils apparaissent dans l'interface mais ne déclenchent aucune installation.

Au lancement normal, les gestionnaires manquants sont bootstrappés avant le plan des outils : Homebrew sur macOS, Scoop sur Windows et `pip` via `ensurepip` lorsque Python est disponible. `winget` dépend de l’App Installer de Microsoft et les gestionnaires Linux natifs dépendent de la distribution : s'ils sont absents, l'outil les signale sans tenter une installation destructive. Le manifeste par défaut est intégré au binaire ; un fichier `tools.json` présent dans le répertoire courant peut le remplacer. `--list` reste toujours sans effet de bord.

## Utilisation

```sh
go run ./cmd/cross-tools
```

Dans la TUI, l'écran affiche uniquement le pack compatible avec l'OS. Il n'y a pas de navigation individuelle :

- `entrée` : installer le pack complet
- `q` : quitter

Les commandes d'installation sont exécutées avec le terminal rendu temporairement à Homebrew, `sudo` ou au gestionnaire concerné : les demandes de mot de passe restent donc interactives et visibles.

Une erreur sur un outil n'arrête pas le pack : les autres installations continuent et le bilan final liste les outils en échec.

Options utiles :

```sh
# Voir les commandes qui seraient exécutées
go run ./cmd/cross-tools --list --dry-run

# Installer tout sans interface interactive
go run ./cmd/cross-tools --yes

# Utiliser un manifeste personnalisé
go run ./cmd/cross-tools --manifest ./mon-manifeste.json

# Vérifier le plan d'une autre plateforme sans l'installer
go run ./cmd/cross-tools --os linux --list --dry-run

# Désactiver le bootstrap automatique des gestionnaires
go run ./cmd/cross-tools --bootstrap=false
```

## Manifeste

Le manifeste par défaut est intégré au binaire et peut être remplacé avec `--manifest`. Chaque outil peut déclarer des alternatives par OS, dans l'ordre de préférence :

```json
{
  "version": 1,
  "tools": [
    {
      "name": "Mon outil",
      "category": "Ma catégorie",
      "description": "Description affichée dans le manifeste",
      "packages": {
        "linux": [
          {"manager": "apt", "name": "mon-paquet"},
          {"manager": "dnf", "name": "mon-paquet"}
        ],
        "darwin": [
          {"manager": "brew", "name": "mon-paquet", "options": ["--cask"]}
        ],
        "windows": [
          {"manager": "winget", "name": "Vendor.MonOutil"}
        ]
      }
    }
  ]
}
```

Les gestionnaires pris en charge sont `apt`, `dnf`, `pacman`, `snap`, `brew`, `scoop`, `winget`, `pip`, `xcode-select`, `script` et `builtin`. Les entrées `all` peuvent servir de repli commun lorsque le nom du paquet est identique. Le gestionnaire `script` exécute explicitement `command` avec `args` : utilisez `--dry-run` pour inspecter ces commandes avant installation.

## Vérification

```sh
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/cross-tools
```

## Releases GitHub

Chaque push sur `main` ou pull request lance les tests sur Linux, macOS et Windows. Le détecteur de race est exécuté sur Linux.

Pour vérifier une vraie installation avant une release, ouvrir `Actions` puis `Installation integration` et lancer le workflow. Par défaut, il installe un petit smoke pack réel sur les trois OS. L'option `full_pack` permet de tester le manifeste complet ; elle est plus longue et dépend des dépôts tiers, des DMG et des permissions de chaque runner.

Pour créer une release, pousser un tag semver :

```sh
git tag v0.1.0
git push origin v0.1.0
```

La GitHub Action reteste les trois OS avant de publier les binaires suivants :

- `cross-tools-linux-amd64`
- `cross-tools-linux-arm64`
- `cross-tools-darwin-amd64`
- `cross-tools-darwin-arm64`
- `cross-tools-windows-amd64.exe`
- `tools.json` (surcharge facultative)

`tools.json` reste publié avec les binaires pour permettre une surcharge ou une modification du manifeste ; le binaire fonctionne aussi sans ce fichier.
