# Промпты для разработки kompoze

> 10 самодостаточных промптов для пошаговой разработки CLI-инструмента.
> Выполняй последовательно. После каждого: `go build ./...` и `go test ./...` должны проходить.

---

## ПРОМПТ 1: Инициализация проекта и структура

```text
# Роль
Ты — senior Go-разработчик с 10+ лет опыта в CLI-инструментах и Kubernetes ecosystem.

# Задача
Инициализируй Go-проект "kompoze" — CLI-инструмент для конвертации docker-compose.yml в production-ready Kubernetes манифесты.

# Что нужно сделать
1. Инициализируй Go module: `go mod init github.com/compositor/kompoze`
2. Создай структуру проекта:

kompoze/
├── cmd/
│   └── root.go              # Cobra root command
│   └── convert.go            # Cobra convert subcommand
│   └── version.go            # версия
├── internal/
│   ├── parser/
│   │   └── compose.go        # парсинг docker-compose.yml
│   │   └── compose_test.go
│   │   └── types.go          # Go-структуры для docker-compose
│   ├── converter/
│   │   └── converter.go      # основная логика конвертации
│   │   └── converter_test.go
│   │   └── deployment.go     # генерация Deployment
│   │   └── service.go        # генерация Service
│   │   └── configmap.go      # генерация ConfigMap
│   │   └── pvc.go            # генерация PVC
│   │   └── ingress.go        # генерация Ingress
│   │   └── hpa.go            # генерация HPA
│   │   └── pdb.go            # генерация PDB
│   │   └── networkpolicy.go  # генерация NetworkPolicy
│   ├── wizard/
│   │   └── wizard.go         # Bubble Tea TUI wizard
│   ├── helm/
│   │   └── generator.go      # генерация Helm chart
│   ├── kustomize/
│   │   └── generator.go      # генерация Kustomize overlays
│   ├── validator/
│   │   └── validator.go      # валидация через kubeconform
│   └── output/
│       └── writer.go         # запись YAML-файлов на диск
├── testdata/
│   ├── simple-compose.yml    # минимальный compose
│   ├── full-compose.yml      # compose со всеми фичами
│   └── wordpress-compose.yml # реальный пример
├── main.go
├── Makefile
├── .goreleaser.yml
└── .gitignore

3. Установи зависимости:
   - github.com/spf13/cobra (CLI)
   - gopkg.in/yaml.v3 (YAML парсинг)
   - k8s.io/api, k8s.io/apimachinery (K8s типы — используй официальные Go-структуры)
   - sigs.k8s.io/yaml (сериализация K8s объектов в YAML)

4. Реализуй базовый CLI:
   - `kompoze` — показывает help
   - `kompoze convert <file> -o <dir>` — заглушка с сообщением
   - `kompoze version` — выводит версию

5. Создай Makefile с таргетами: build, test, lint, install

6. Создай .gitignore для Go-проекта

# Контекст
- Это open-source CLI-инструмент, конкурент Kompose (CNCF, 9K stars)
- Наше преимущество: production-grade output с best practices
- Модуль называется: github.com/compositor/kompoze
- Минимальная версия Go: 1.22

# Правила
- Используй стандартную Go project layout (cmd/, internal/)
- Все внутренние пакеты в internal/ — они не экспортируются
- Cobra для CLI, НЕ пиши свой парсер аргументов
- Используй официальные K8s Go-типы (k8s.io/api/apps/v1, k8s.io/api/core/v1 и т.д.), НЕ генерируй YAML строками
- Код должен компилироваться и запускаться после выполнения этого промпта
- Не добавляй фичи сверх описанного — только скелет

# Формат вывода
Рабочий Go-проект со всеми файлами. Каждый файл с правильными package declarations и imports.
```

---

## ПРОМПТ 2: Парсер docker-compose v3.8+

```text
# Роль
Ты — senior Go-разработчик, эксперт по Docker Compose спецификации и YAML-парсингу.

# Задача
Реализуй полный парсер docker-compose.yml v3.8+ для проекта kompoze.

# Контекст
Проект kompoze конвертирует docker-compose.yml в Kubernetes манифесты. Структура проекта уже создана. Парсер находится в `internal/parser/`. Используется `gopkg.in/yaml.v3` для парсинга.

# Что нужно сделать

## 1. Go-структуры (internal/parser/types.go)
Создай Go-структуры, покрывающие docker-compose v3.8+ спецификацию:

type ComposeFile struct {
    Version  string                    `yaml:"version"`
    Services map[string]ServiceConfig  `yaml:"services"`
    Volumes  map[string]VolumeConfig   `yaml:"volumes,omitempty"`
    Networks map[string]NetworkConfig  `yaml:"networks,omitempty"`
    Secrets  map[string]SecretConfig   `yaml:"secrets,omitempty"`
    Configs  map[string]ConfigConfig   `yaml:"configs,omitempty"`
}

Для ServiceConfig поддержи ВСЕ ключевые поля:
- image, build, container_name, command, entrypoint
- ports (short и long syntax: "8080:80" и {target: 80, published: 8080, protocol: tcp})
- volumes (short и long syntax: "./data:/app/data" и {type: bind, source: ..., target: ...})
- environment (список "KEY=VALUE" и map {KEY: VALUE})
- env_file
- depends_on (список и расширенный с condition)
- healthcheck (test, interval, timeout, retries, start_period)
- deploy (replicas, resources.limits, resources.reservations, restart_policy, placement)
- labels, networks, expose, restart, logging
- user, working_dir, stdin_open, tty, privileged, read_only
- cap_add, cap_drop, security_opt, sysctls
- extra_hosts, dns, dns_search

## 2. Парсер (internal/parser/compose.go)
func ParseComposeFile(path string) (*ComposeFile, error)
func ParseComposeBytes(data []byte) (*ComposeFile, error)

Парсер должен:
- Читать и десериализовать YAML
- Валидировать version >= 3.8
- Нормализовать данные: привести ports, volumes, environment к каноничному формату
- Обрабатывать переменные окружения ${VAR:-default} — заменять из os.Environ или использовать default
- Возвращать понятные ошибки с указанием строки/сервиса

## 3. Тесты (internal/parser/compose_test.go)
Напиши table-driven тесты покрывающие:
- Минимальный compose (1 сервис, только image)
- Полный compose (все поля)
- Парсинг портов: "8080:80", "80", "8080:80/udp", long syntax
- Парсинг volumes: named, bind, short, long syntax
- Парсинг environment: list и map формат
- Парсинг depends_on: list и extended формат
- Парсинг healthcheck
- Парсинг deploy с ресурсами
- Ошибки: невалидный YAML, неподдерживаемая версия, пустой файл

## 4. Тестовые fixtures (testdata/)
Создай 3 тестовых docker-compose.yml:
- `testdata/simple-compose.yml` — nginx + redis, минимальная конфигурация
- `testdata/full-compose.yml` — 3 сервиса (api, web, db) со ВСЕМИ поддерживаемыми полями
- `testdata/wordpress-compose.yml` — реальный WordPress + MySQL compose

# Правила
- Используй yaml.v3, НЕ yaml.v2
- Для полей с двойным синтаксисом (ports, volumes, environment) используй custom UnmarshalYAML
- environment: всегда нормализуй в map[string]string
- ports: всегда нормализуй в []PortConfig{HostPort, ContainerPort, Protocol}
- volumes: всегда нормализуй в []VolumeMount{Type, Source, Target, ReadOnly}
- Все ошибки оборачивай с контекстом через fmt.Errorf("parsing service %q: %w", name, err)
- Тесты должны проходить: `go test ./internal/parser/...`
- НЕ используй внешние библиотеки для парсинга compose (типа compose-go от Docker) — пишем свой парсер

# Формат вывода
Готовые Go-файлы: types.go, compose.go, compose_test.go и тестовые fixtures. Все тесты проходят.
```

---

## ПРОМПТ 3: Конвертер — Deployment + Service + ConfigMap

```text
# Роль
Ты — senior Go-разработчик и Kubernetes expert, специализирующийся на production-grade манифестах.

# Задача
Реализуй core-конвертер: преобразование распарсенного docker-compose в Kubernetes Deployment, Service и ConfigMap манифесты с production best practices.

# Контекст
Проект kompoze. Парсер docker-compose уже реализован в `internal/parser/`. Типы: `parser.ComposeFile`, `parser.ServiceConfig`. Конвертер — в `internal/converter/`.

Используем официальные K8s Go-типы:
- k8s.io/api/apps/v1 (Deployment)
- k8s.io/api/core/v1 (Service, ConfigMap, PVC, etc.)
- k8s.io/apimachinery/pkg/apis/meta/v1 (ObjectMeta)

# Что нужно сделать

## 1. Основной конвертер (internal/converter/converter.go)
type ConvertOptions struct {
    OutputDir     string
    Namespace     string   // default: "default"
    AppName       string   // из имени compose-файла или --app-name
    AddProbes     bool     // default: true
    AddResources  bool     // default: true
    AddSecurity   bool     // default: true
}

type ConvertResult struct {
    Deployments    []appsv1.Deployment
    Services       []corev1.Service
    ConfigMaps     []corev1.ConfigMap
    PVCs           []corev1.PersistentVolumeClaim
    // ... остальные ресурсы добавятся позже
}

func Convert(compose *parser.ComposeFile, opts ConvertOptions) (*ConvertResult, error)

## 2. Deployment генератор (internal/converter/deployment.go)

Для каждого сервиса в compose генерируй Deployment с:

**Обязательно (всегда):**
- apiVersion: apps/v1, kind: Deployment
- metadata: name, namespace, labels (app.kubernetes.io/name, app.kubernetes.io/part-of, app.kubernetes.io/managed-by: kompoze)
- spec.replicas: из deploy.replicas или 1
- spec.selector.matchLabels
- Pod template с labels
- Container: name, image, ports, env, command, args, volumeMounts

**Smart defaults (если AddProbes=true):**
- Если есть healthcheck в compose → мапь в livenessProbe + readinessProbe
- Если НЕТ healthcheck → генерируй дефолтные probes на основе портов:
  - HTTP порт (80, 8080, 3000, 5000, 8000) → httpGet probe на /healthz или /
  - TCP порт → tcpSocket probe
  - Без портов → НЕ добавляй probes

**Smart defaults (если AddResources=true):**
- Если есть deploy.resources в compose → используй их
- Если НЕТ → добавь дефолтные:
  - requests: cpu=100m, memory=128Mi
  - limits: cpu=500m, memory=256Mi

**Smart defaults (если AddSecurity=true):**
- securityContext на pod уровне: runAsNonRoot=true, fsGroup=1000
- securityContext на container уровне: allowPrivilegeEscalation=false, readOnlyRootFilesystem=true (если не privileged в compose), capabilities.drop=["ALL"]
- Если в compose есть cap_add/cap_drop — мапь их

**Маппинг depends_on:**
- depends_on сервисов → initContainers с busybox, ждущие доступности зависимости через `nc -z <service> <port>`

## 3. Service генератор (internal/converter/service.go)

Для каждого сервиса с ports:
- ClusterIP Service для внутренних портов
- Если published port отличается от target — используй nodePort или указывай в Service

Логика выбора типа:
- ports без host mapping → ClusterIP
- ports с host mapping → сохраняй маппинг, комментарий что для production лучше Ingress

## 4. ConfigMap генератор (internal/converter/configmap.go)

- environment переменные → ConfigMap (не-секретные)
- Если имя переменной содержит PASSWORD, SECRET, TOKEN, KEY, CREDENTIALS → НЕ клади в ConfigMap, пометь как Secret reference
- env_file → читай файл, создавай ConfigMap

## 5. Сериализация в YAML (internal/output/writer.go)
func WriteManifests(result *ConvertResult, outputDir string) error
- Каждый ресурс в отдельный файл: `<service-name>-deployment.yaml`, `<service-name>-service.yaml`
- Или все в один файл разделённые `---`
- Используй sigs.k8s.io/yaml для сериализации K8s объектов
- Добавляй комментарий в начало: `# Generated by kompoze - Do not edit manually`

## 6. Тесты (internal/converter/converter_test.go)

Table-driven тесты:
- Минимальный сервис (только image) → корректный Deployment
- Сервис с портами → Deployment + Service
- Сервис с environment → Deployment + ConfigMap
- Сервис с healthcheck → probes маппятся корректно
- Сервис с deploy.resources → ресурсы маппятся
- Сервис с depends_on → initContainers генерируются
- Сервис с sensitive env vars → разделение ConfigMap/Secret ref
- Полный compose файл → все ресурсы генерируются

# Правила
- Используй ТОЛЬКО официальные K8s Go-типы, НЕ строковую конкатенацию YAML
- Labels должны следовать конвенции app.kubernetes.io/*
- Все генерируемые ресурсы должны проходить валидацию kubeconform
- Дефолтные resource limits разумные для development (не production)
- Security context по умолчанию restrictive — пользователь может отключить флагом
- Код должен быть покрыт тестами >= 80%

# Формат вывода
Готовые Go-файлы конвертера. Тесты проходят. Output валидный K8s YAML.
```

---

## ПРОМПТ 4: PVC, Volumes и полный маппинг

```text
# Роль
Ты — senior Go-разработчик и Kubernetes storage expert.

# Задача
Реализуй маппинг Docker Compose volumes в Kubernetes PersistentVolumeClaim (PVC) и volumeMounts для проекта kompoze.

# Контекст
Проект kompoze. Парсер и базовый конвертер (Deployment, Service, ConfigMap) уже реализованы. Нужно добавить поддержку volumes. Файлы: `internal/converter/pvc.go` и обновление `deployment.go`.

# Что нужно сделать

## 1. PVC генератор (internal/converter/pvc.go)

Маппинг compose volumes → K8s:

| Compose Volume | K8s Resource |
|---|---|
| Named volume (`db-data:/var/lib/mysql`) | PVC + volumeMount |
| Bind mount (`./config:/app/config`) | Пропускай с warning (не портируется) или ConfigMap если файл |
| tmpfs (`type: tmpfs`) | emptyDir с medium: Memory |
| Top-level volumes с driver | PVC со StorageClassName |

Для PVC:
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: <service>-<volume-name>
  labels: ...
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1Gi  # дефолт, переопределяется в wizard

## 2. Обновление Deployment

В pod template добавляй:
- `volumes:` — PVC → persistentVolumeClaim, tmpfs → emptyDir
- Container `volumeMounts:` — с правильными mountPath и readOnly

## 3. Bind mount → ConfigMap (для файлов конфигурации)

Если bind mount указывает на один файл (не директорию):
- Прочитай файл
- Создай ConfigMap с содержимым
- Замонтируй как subPath volume

Если bind mount указывает на директорию:
- Выведи warning: "Bind mount './data:/app/data' не может быть автоматически сконвертирован. Используйте PVC или ConfigMap."

## 4. Тесты

- Named volume → PVC + volumeMount
- Multiple volumes → множественные PVC
- tmpfs → emptyDir
- Bind mount файл → ConfigMap
- Bind mount директория → warning
- ReadOnly volume → readOnly: true в volumeMount

# Правила
- Дефолтный размер PVC: 1Gi (позже wizard позволит менять)
- AccessMode: ReadWriteOnce по умолчанию; ReadWriteMany если volume шарится между сервисами
- Имена PVC: lowercase, kebab-case, максимум 63 символа
- Не теряй данные — если не можешь сконвертировать volume, выведи warning, не молчи

# Формат вывода
Файлы pvc.go, обновлённый deployment.go, тесты.
```

---

## ПРОМПТ 5: Wizard-режим (Bubble Tea TUI)

```text
# Роль
Ты — senior Go-разработчик с опытом создания TUI-приложений на Bubble Tea и deep knowledge Kubernetes best practices.

# Задача
Реализуй интерактивный wizard-режим для kompoze, который проводит пользователя через настройку конвертации с умными рекомендациями.

# Контекст
Проект kompoze конвертирует docker-compose в K8s. Парсер и конвертер уже работают. Wizard запускается через `kompoze convert --wizard docker-compose.yml`. Используем Bubble Tea (github.com/charmbracelet/bubbletea) + Lip Gloss для стилизации.

# Что нужно сделать

## 1. Wizard flow (internal/wizard/wizard.go)

Wizard анализирует compose-файл и задаёт контекстные вопросы:

### Шаг 1: Общие настройки
Kompoze Wizard
Анализирую docker-compose.yml... Найдено 3 сервиса: api, web, db

? Namespace для деплоя: [default] ___
? Формат вывода:
  > Kubernetes manifests
    Helm chart
    Kustomize (base + overlays)

### Шаг 2: Для каждого сервиса — контекстные вопросы

Wizard анализирует сервис и задаёт умные вопросы:

── Сервис: web (nginx:1.25) ──

? Тип сервиса обнаружен: web-server. Добавить Ingress? [Y/n]
? Hostname для Ingress: [web.example.com] ___
? Добавить TLS (cert-manager)? [Y/n]
? Replicas: [2] ___
? Добавить HPA (auto-scaling)? [Y/n]
  Min replicas: [2] ___
  Max replicas: [10] ___
  Target CPU%: [70] ___
? Resource limits:
  CPU request: [100m] ___  CPU limit: [500m] ___
  Memory request: [128Mi] ___  Memory limit: [256Mi] ___

── Сервис: db (postgres:15) ──

? Тип сервиса обнаружен: database. Рекомендуем StatefulSet вместо Deployment.
  Использовать StatefulSet? [Y/n]
? Размер PVC для данных: [10Gi] ___
? Добавить PodDisruptionBudget? [Y/n]
  minAvailable: [1] ___
? Environment переменные с SECRET/PASSWORD обнаружены.
  Создать Kubernetes Secret? [Y/n]

### Шаг 3: Smart detection

Wizard автоматически определяет тип сервиса по image name:
- `nginx`, `httpd`, `traefik`, `caddy` → web-server → предлагает Ingress
- `postgres`, `mysql`, `mongo`, `redis`, `elasticsearch` → database/cache → предлагает StatefulSet, PDB
- `node`, `python`, `golang`, `java` → app-server → предлагает HPA
- Неизвестный image → generic, базовые вопросы

### Шаг 4: Summary и подтверждение
Summary table с Service, Kind, Replicas, Ingress, HPA, PDB для каждого сервиса.
? Сгенерировать манифесты? [Y/n]

## 2. Модель данных wizard

type WizardConfig struct {
    Namespace    string
    OutputFormat string // "manifests" | "helm" | "kustomize"
    Services     map[string]ServiceWizardConfig
}

type ServiceWizardConfig struct {
    Kind         string // "Deployment" | "StatefulSet"
    Replicas     int32
    AddIngress   bool
    IngressHost  string
    AddTLS       bool
    AddHPA       bool
    HPAMin       int32
    HPAMax       int32
    HPATargetCPU int32
    AddPDB       bool
    PDBMinAvail  int32
    Resources    ResourceConfig
    PVCSize      string
    CreateSecret bool
}

## 3. Интеграция с конвертером

WizardConfig передаётся в converter.Convert() и переопределяет дефолты.

## 4. UX требования

- Дефолтные значения в квадратных скобках — Enter принимает дефолт
- Стрелки вверх/вниз для выбора из списка
- Цветовая схема: зелёный для подтверждений, жёлтый для рекомендаций, красный для warnings
- q или Ctrl+C для выхода в любой момент
- Progress bar при генерации

# Правила
- Используй Bubble Tea, НЕ пиши свой TUI с нуля
- Используй Lip Gloss для стилизации (github.com/charmbracelet/lipgloss)
- Wizard должен быть optional — без --wizard конвертация работает с дефолтами
- Все дефолты должны быть разумными — пользователь может просто нажимать Enter
- Wizard должен работать в обычном терминале (без mouse support requirement)
- Не блокируй CI/CD — если stdin не TTY, пропускай wizard с warning

# Формат вывода
Файлы wizard/wizard.go, wizard/styles.go, wizard/detection.go (определение типа сервиса). Обновлённый cmd/convert.go с --wizard флагом.
```

---

## ПРОМПТ 6: Ingress, HPA, PDB, NetworkPolicy

```text
# Роль
Ты — senior Go-разработчик и Kubernetes platform engineer с опытом production-grade кластеров.

# Задача
Реализуй генерацию Ingress, HorizontalPodAutoscaler (HPA), PodDisruptionBudget (PDB), ServiceAccount и NetworkPolicy для проекта kompoze.

# Контекст
Проект kompoze. Конвертер генерирует Deployment, Service, ConfigMap, PVC. Нужно добавить оставшиеся production-critical ресурсы.

# Что нужно сделать

## 1. Ingress (internal/converter/ingress.go)
Генерируй Ingress для сервисов с published портами на HTTP (80, 443, 8080, 3000, 5000, 8000):
- apiVersion: networking.k8s.io/v1
- ingressClassName: nginx (дефолт)
- TLS с cert-manager annotation если включен
- host: <service>.example.com или из wizard

## 2. HPA (internal/converter/hpa.go)
Генерируй HPA для не-database сервисов:
- apiVersion: autoscaling/v2
- minReplicas: 2, maxReplicas: 10
- target CPU utilization: 70%

## 3. PDB (internal/converter/pdb.go)
Генерируй PDB для сервисов с replicas > 1:
- apiVersion: policy/v1
- minAvailable: 1 или "50%" для сервисов с replicas > 2

## 4. ServiceAccount (internal/converter/serviceaccount.go)
Для каждого сервиса:
- automountServiceAccountToken: false (security best practice)
- Готово для IRSA/Workload Identity annotations

## 5. NetworkPolicy (internal/converter/networkpolicy.go)
На основе depends_on и ports:
- Разрешай ingress только от зависимых сервисов
- Разрешай egress только к зависимостям
- Default deny для остального

## 6. Тесты для каждого генератора

# Правила
- Используй правильные apiVersion для каждого ресурса
- Ingress генерируется ТОЛЬКО если есть published HTTP порт или wizard включил
- HPA НЕ генерируется для databases (postgres, mysql, mongo, redis)
- PDB генерируется ТОЛЬКО если replicas > 1
- NetworkPolicy должна быть опциональной (--network-policy флаг)
- automountServiceAccountToken: false по умолчанию

# Формат вывода
Go-файлы для каждого генератора + тесты. Обновлённый ConvertResult.
```

---

## ПРОМПТ 7: Helm Chart Output

```text
# Роль
Ты — senior Go-разработчик и Helm expert с опытом создания production Helm charts.

# Задача
Реализуй генерацию Helm chart из docker-compose для проекта kompoze. Запускается через `kompoze convert --helm`.

# Контекст
Проект kompoze. Конвертер уже генерирует все K8s ресурсы. Нужно обернуть output в Helm chart структуру.

# Что нужно сделать

## 1. Helm chart структура (internal/helm/generator.go)
Генерируй: Chart.yaml, values.yaml, templates/ (для каждого ресурса), _helpers.tpl, NOTES.txt

## 2. values.yaml
Вынеси ВСЕ конфигурируемые параметры: image, replicas, resources, service, ingress, hpa, pdb, env, secrets, persistence — для каждого сервиса.

## 3. Templates с Helm templating
Каждый template использует {{ .Values.<service>.* }} вместо hardcoded значений.

## 4. _helpers.tpl
Стандартные helpers: name, fullname, labels, selectorLabels.

## 5. NOTES.txt
Post-install instructions с URLs.

# Правила
- Следуй Helm best practices
- values.yaml задокументирован комментариями
- Все ресурсы условные через `if .Values.<service>.enabled`
- Secrets помечены "# CHANGE_ME — do not commit real secrets"
- Chart должен проходить `helm lint` и `helm template`

# Формат вывода
Go-файл generator.go, генерирующий полную Helm chart структуру.
```

---

## ПРОМПТ 8: Kustomize Output

```text
# Роль
Ты — senior Go-разработчик с опытом Kustomize и multi-environment деплоев.

# Задача
Реализуй генерацию Kustomize структуры (base + overlays для dev/staging/prod) для проекта kompoze. Запускается через `kompoze convert --kustomize`.

# Контекст
Проект kompoze. Конвертер генерирует все K8s ресурсы. Нужна Kustomize-совместимая структура.

# Что нужно сделать

## 1. Структура: base/ + overlays/dev, overlays/staging, overlays/prod

## 2. Base kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1 с commonLabels и resources.

## 3. Overlays per environment
- Dev: replicas=1, minimal resources, no ingress/HPA/PDB, namespace=dev
- Staging: replicas=2, medium resources, Ingress with staging host, namespace=staging
- Prod: replicas=3+, full resources, Ingress+TLS, HPA, PDB, namespace=production

## 4. Strategic merge patches для overlays

# Правила
- Base без environment-specific конфигурации
- Overlays используют patches, НЕ дублируют ресурсы
- Namespace в overlay kustomization.yaml, НЕ в base
- `kustomize build overlays/prod` должен работать

# Формат вывода
Go-файл generator.go, генерирующий полную Kustomize структуру.
```

---

## ПРОМПТ 9: Валидация и CLI интеграция

```text
# Роль
Ты — senior Go-разработчик, expert в CLI tooling и Kubernetes manifest validation.

# Задача
Реализуй встроенную валидацию сгенерированных манифестов и финализируй CLI для проекта kompoze.

# Контекст
Проект kompoze. Все генераторы реализованы. Нужно: валидация, полировка CLI, обработка ошибок.

# Что нужно сделать

## 1. Валидатор (internal/validator/validator.go)
Встроенные проверки: required fields, image tag != "latest", resource limits, probes, port matching, labels convention.
Kubeconform: опциональная внешняя валидация если kubeconform в PATH.

## 2. CLI финализация (cmd/convert.go)
Все флаги: -o, -n, --app-name, --helm, --kustomize, --wizard, --validate, --strict, --no-probes, --no-resources, --no-security, --no-network-policy, --single-file, -q, -v, --dry-run

## 3. Output formatting
Прогресс с checkmarks, warnings, summary.

## 4. Error handling
Понятные ошибки, автосоздание директорий, предупреждение о перезаписи.

## 5. Integration тесты
E2E: Parse → Convert → Write → Validate.

# Правила
- Валидация по умолчанию включена, non-blocking (warnings)
- --strict: warnings → errors
- Exit codes: 0 success, 1 error, 2 validation warnings (strict)
- Цветной output (github.com/fatih/color или lipgloss)
- --dry-run выводит YAML в stdout

# Формат вывода
Обновлённые validator.go, cmd/convert.go, integration_test.go.
```

---

## ПРОМПТ 10: Тесты, CI/CD, README, Release

```text
# Роль
Ты — senior Go-разработчик с опытом open-source проектов, CI/CD и developer marketing.

# Задача
Финализируй проект kompoze: полное тестовое покрытие, CI/CD pipeline, README, GoReleaser.

# Контекст
Проект kompoze полностью реализован. Нужно подготовить к open-source релизу.

# Что нужно сделать

## 1. Тестовое покрытие
- Unit: каждый пакет >= 80%, edge cases
- Integration: WordPress, Django, Laravel, Microservices compose files
- Golden file тесты: testdata/golden/, `go test -update`
- Тестовые compose файлы: simple, wordpress, django, laravel, microservices

## 2. CI/CD (.github/workflows/)
- ci.yml: test (go test -race), build, integration, lint (golangci-lint)
- release.yml: goreleaser on tag push v*

## 3. GoReleaser (.goreleaser.yml)
Cross-compile: linux/darwin amd64/arm64, windows/amd64. Homebrew tap.

## 4. README.md
- Comparison table: kompoze vs kompose (resource limits, probes, security, helm, kustomize, wizard, validation)
- Quick Start, Installation, Usage Examples, Contributing, License (MIT)

## 5. Makefile
build, test, test-integration, lint, coverage, install, clean, release

# Правила
- README должен "продавать" проект
- CI на каждый PR
- Тесты НЕ требуют K8s кластер
- Golden files обновляются вручную (-update flag)
- License: MIT

# Формат вывода
Тесты, CI workflows, .goreleaser.yml, README.md, Makefile, LICENSE.
```

---

## Порядок выполнения

| # | Промпт | Неделя | Зависимости |
|---|--------|--------|-------------|
| 1 | Инициализация проекта | 1 | — |
| 2 | Парсер docker-compose | 1 | Промпт 1 |
| 3 | Конвертер: Deployment + Service + ConfigMap | 1-2 | Промпт 2 |
| 4 | PVC и Volumes | 2 | Промпт 3 |
| 5 | Wizard-режим | 3 | Промпт 3 |
| 6 | Ingress, HPA, PDB, NetworkPolicy | 3 | Промпт 3 |
| 7 | Helm Chart Output | 4 | Промпт 6 |
| 8 | Kustomize Output | 4 | Промпт 6 |
| 9 | Валидация и CLI | 4-5 | Промпты 7, 8 |
| 10 | Тесты, CI/CD, README | 5 | Все |

## Верификация
- После каждого промпта: `go build ./...` и `go test ./...` должны проходить
- После промпта 3: `kompoze convert testdata/simple-compose.yml -o /tmp/k8s` генерирует валидный YAML
- После промпта 7: `helm lint /tmp/helm-output` проходит
- После промпта 8: `kustomize build /tmp/kustomize-output/overlays/prod` проходит
- После промпта 10: все CI checks зелёные, README отрендерен корректно
