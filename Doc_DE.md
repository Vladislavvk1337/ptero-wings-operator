# ptero-wings-operator – Architektur und Betriebsdoku

## Überblick

Das Projekt trennt die frühere Wings-Logik jetzt in drei klar abgegrenzte Schichten:

1. **ptero-wings-gateway**
   - äußerer HTTP-/WebSocket-Einstiegspunkt
   - Authentifizierung via Token oder JWT
   - API-Bridge für das Panel
2. **ptero-wings-controller**
   - Geschäftslogik
   - direkter Kubernetes-Client
   - Startup-Command-Mapping
   - NodePort-Bündelung
   - Metriken und optionale Panel-REST-Integration
3. **ptero-wings-operator**
   - CRDs `GameServer` und `GameServerClass`
   - Reconciliation von PVC, Service, StatefulSet und Secret
   - Finalizer, Conditions und Statuspflege

## Request-Flow

1. Das externe Panel spricht mit dem Gateway.
2. Das Gateway prüft Authentifizierung und delegiert an den Controller.
3. Der Controller erstellt oder ändert `GameServer`-Ressourcen über die Kubernetes-API.
4. Der Operator reconciled die CRs in Cluster-Ressourcen.
5. Status, Logs, Konsole und Ressourcenwerte laufen zurück über Controller und Gateway.

## GameServer- und Kubernetes-Ressourcen

Ein `GameServer` beschreibt den gewünschten Zustand eines einzelnen Spielservers.
Der Operator erzeugt daraus typischerweise:

- ein `Secret` für Metadaten,
- ein dediziertes `PersistentVolumeClaim`,
- einen `Service` (meist `NodePort`),
- ein `StatefulSet` mit dem eigentlichen Game-Container.

`GameServerClass` dient als Default-/Template-Schicht für Image, Ressourcen, Storage, Netzwerk und Startup-Defaults.

## Startup-Command-Mapping

Der Controller verarbeitet Startup-Strings aus dem Panel über:

- `spec.runtime.startup.raw`
- `spec.runtime.startup.variables`

Beispiel:

```yaml
runtime:
  startup:
    raw: java -Xmx{{SERVER_MEMORY}}M -jar {{SERVER_JARFILE}}
    variables:
      SERVER_MEMORY: "1024"
      SERVER_JARFILE: server.jar
```

Der Controller übersetzt das sauber in:

- `spec.game.command`
- `spec.game.args`

Dadurch bleibt die Container-Definition Kubernetes-nativ und testbar.

## NodePort-Bündelung

Für zusammenhängende Portreservierungen gibt es `spec.network.nodePortAllocation`.

Beispiel:

```yaml
network:
  serviceType: NodePort
  nodePortAllocation:
    startPort: 30005
    count: 5
    containerStartPort: 25565
```

In dieser Implementierung bedeutet `count`, dass ab `startPort` eine inklusive Portgruppe reserviert wird. Das Beispiel reserviert also `30005-30010`.
Wenn `startPort` leer bleibt, sucht der Controller automatisch den nächsten freien Bereich innerhalb der konfigurierten NodePort-Range.

## Metrics-Flow

Der Controller sammelt serverbezogene Werte und liefert sie im Wings-ähnlichen Format zurück:

- `cpu_absolute`
- `memory_bytes`
- `disk_bytes`
- `network_rx_bytes`
- `network_tx_bytes`
- `uptime_seconds`

Zusätzlich können Cluster-Werte wie Node-Anzahl, Pod-Anzahl, CPU- und RAM-Kapazität mitgegeben werden.
Wenn `PANEL_URL` und `PANEL_TOKEN` gesetzt sind, kann der Controller diese Daten zusätzlich aktiv an das Panel senden.

## Longhorn / Storage

Die Storage-Schicht bleibt CRD-basiert. Der Operator verwaltet weiterhin dedizierte PVCs pro `GameServer`.
Longhorn oder andere CSI-/Storage-Systeme können über StorageClass, Backup-/Snapshot-Policies und externe Automatisierung integriert werden.

## Deployment-Hinweis

Standardmäßig wird der Controller-Code aktuell direkt im Gateway-Prozess verwendet. Dadurch ist die logische Schichtentrennung bereits im Code umgesetzt. Beispielmanifeste für eine spätere physische Trennung als eigener Controller-Deployment liegen unter `examples/controller/`.
