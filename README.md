# SupTunnels

Универсальная система реверс-прокси туннелирования с веб-интерфейсом и TUI на Go.

## Возможности
- Проксирование TCP и UDP трафика.
- Режим "both" (TCP и UDP на одном порту).
- Мультиплексирование через `yamux`.
- Веб-интерфейс для управления туннелями и просмотра статистики.
- Красивый TUI на базе `bubbletea` и `lipgloss`.
- Автоопределение протоколов.

## Сборка
Требуется Go 1.21+.

```bash
make build
```

## Запуск

### Сервер (на машине с публичным IP)
```bash
./suptunnels-server --port 8080 --secret mysecret --public-ip 1.2.3.4
```

### Клиент (за NAT)
```bash
./suptunnels-client --server 1.2.3.4:8080 --secret mysecret
```

## Настройка
Конфигурация хранится в `~/.suptunnels/config.yaml`.
Пример:
```yaml
server:
  listen_addr: ":8080"
  secret: "mysecret"
tunnels:
  - id: "mc-java"
    name: "Minecraft Java"
    external_port: 25565
    internal_port: 25565
    type: "tcp"
    enabled: true
```
