# Ревью и исправления tdm-light — 1 ноября 2026

Исправления согласованы пользователем и внесены. Для Discord и скрытого ввода добавлены закреплённые зависимости gorilla/websocket, x/term и косвенная x/sys.

## Проверка комплекта VPS перед публикацией

Комплект пересобран скриптом build-vps.ps1 для Linux amd64 и arm64 с CGO_ENABLED=0.
Проверены ELF64, архитектуры и отсутствие динамического интерпретатора. Архив
распакован отдельно: ровно семь ожидаемых файлов, хеши совпадают с исходными
членами архива, Linux-скрипты/настройки имеют LF. Личных конфигов, аккаунтов,
логов и кэша в архиве нет. Синтаксис трёх Bash-установщиков проверен Git Bash.
В Docker-установщике исправлена ошибка local file/tmp в одной строке, вызывавшая
unbound variable; сохранение настройки воспроизведено на отдельном тестовом файле.
Docker daemon недоступен, WSL не удалось использовать: реальный запуск ELF
на Ubuntu и запуск systemd/контейнера в этой среде не проверены.

Перед коммитом проверены предлагаемые файлы на совпадение с рабочими Telegram
и Twitch-токенами и длинными паролями: совпадений нет. Локальные данные,
бинарники и готовые архивы исключены из Git. Шаблоны не содержат учётных данных.

## Сводка скорости в Telegram

Команда /status дополнена средней скоростью по свежим окнам активных аккаунтов,
счётчиками «нормально ≥90%», «отстают», «сбор/обновление данных», числом попыток
восстановления скорости за текущий запуск и пятью самыми медленными логинами.
Для каждого среднего явно указано число аккаунтов с пригодными данными.
Источник — существующие наблюдения Inventory за окно 10–15 минут; новых запросов
или фоновых задач нет. До полного окна среднее не показывается. При смене
канала/дропа, выходе из фарма и попытке восстановления замер очищается.
Устаревшие замеры не участвуют в среднем; простаивающие и ошибочные аккаунты
также исключены. Порог сводки 90% не меняет правило автоматического восстановления.

Проверены свежие/устаревшие окна, отсутствие данных, исключение неактивных
аккаунтов, очистка замеров, конкурентные обновления и компактность сообщения
для 500 аккаунтов. Полный набор прошёл 10 повторений; go vet прошёл.

## Дополнение после проверки реального запуска

Расписание просмотра дополнительно разделено: HLS каждые 20 секунд,
beacon по HEARTBEAT_INTERVAL_SEC (не чаще раза в минуту), Inventory по
CAMPAIGN_POLL_INTERVAL_SEC (по умолчанию раз в две минуты). Одна горутина
аккаунта обслуживает отдельные дедлайны; сетевые запросы могут задерживать
их выполнение. Пропущенные слоты не вызывают запросов вдогонку. После
переключения канала расписание beacon сохраняется. Ошибки HLS по-прежнему
вызывают смену канала, даже если beacon до этого проходил.

Тесты независимых частот, пропуска просроченных слотов и существующие
регрессионные сценарии прошли 10 повторений; go vet прошёл. Эффект частоты
HLS требует живого прогона; тесты расписания не подтверждают зачисление времени Twitch.

После 30-минутного прогона добавлен контроль скорости: подтверждённые минуты
за окно 10–15 минут, не менее четырёх наблюдений. Порог — прирост ниже 80%
длительности окна с допуском в одну минуту и округлением ожидаемого значения
вниз. Небольшие положительные прибавки не сбрасывают окно. Данные отсутствуют,
счётчик уменьшился, часы пошли назад или между наблюдениями большая пауза —
окно сбрасывается. После смены канала начинается новое наблюдение.

При медленном начислении проверяются другие доступные каналы, максимум три
попытки открытия. Если замены нет, обновляется просмотр текущего канала.
Если восстановление не удалось, рабочий текущий просмотр продолжается.
Пауза 10–12 минут сохраняется на аккаунте между дропами и обычными повторами;
полный reload менеджера сбрасывает это состояние. Дополнительных регулярных
запросов и горутин нет: используются существующие опросы Inventory.

Тесты медленного положительного прироста, нормального начисления, догоняющего
обновления, неизвестных данных, разрывов наблюдений, ограниченности истории,
паузы восстановления, смены канала и переподключения единственного канала,
а также весь набор прошли 10 повторений. go vet прошёл. Реальный эффект нового
восстановления на скорость начисления ещё требует прогона.

Исправлено чтение личных минут: каталог кампаний объединяется с
`currentUser.inventory.dropCampaignsInProgress` в одном запросе Inventory.
Значения self из каталога больше не принимаются за личные минуты. Нулевой
подтверждённый прогресс отличается от отсутствующих данных; каждый опрос
записывает текущее значение, даже когда оно не изменилось. Команда /logs
показывает последние минуты и время их проверки в начале ответа.

Однократная проверка чтением реального аккаунта вернула 13 минут для четырёх
наград одной кампании и готовые 60/60, 120/120, 180/180, 240/240 для другой.
Heartbeat и получение наград во время этой проверки не выполнялись.

После неудачного восстановления HLS воркер меняет канал даже при успешном
beacon. Проблемный канал исключается на пять минут; при открытии потока
проверяется максимум три кандидата перед обычной паузой повторения. Отмена
запуска больше не порождает ложные предупреждения о переподключении.

Добавлены тесты объединения устаревшего каталога с Inventory, отсутствующих
личных данных, переключения при работающем beacon и постоянном HLS 404,
видимости неизменившихся минут и отмены без лишних запросов.

Структура Inventory сверена с [исходным клиентом TwitchDropsMiner](https://github.com/DevilXD/TwitchDropsMiner/blob/master/twitch.py),
затем проверена реальным запросом. Авторизация и список аккаунтов в отчёт не включены.

## Исправления

- Завершившиеся группы больше не запускаются в непрерывном цикле. Между началами коротких циклов выдерживается campaign_poll_interval_sec, минимум 60 секунд. Во время ожидания reload и завершение процесса обрабатываются сразу.
- Аккаунт с ошибкой авторизации отключается до явного reload. Другие аккаунты продолжают работу.
- Offset Telegram сохраняется между горячими перезагрузками менеджера. Новый polling запускается после завершения предыдущего. Отмена не подтверждает оставшиеся необработанные сообщения пакета. При смене токена создаётся новое состояние polling. Offset не сохраняется между перезапусками всего процесса.
- Программа только набирает время. Готовые дропы учитываются и сообщаются как готовые к ручному получению. Автоматический ClaimDropRewards удалён вместе с ошибочным признанием любого его ответа успешным. Получение, привязка игрового аккаунта и передача награды в игру не выполняются. Один drop ID учитывается один раз за жизнь менеджера.
- Дубли login, включая различия только в регистре, отклоняются. Login нормализуется в нижний регистр. Разрешены латинские буквы, цифры и подчёркивание; пути вроде ../outside не допускаются. Неверные номера групп дают ошибку.
- WebSocket: длина проверяется до выделения памяти; весь фрагментированный текст ограничен 1 МиБ; проверяются флаги кадров и ответ handshake. Upgrade имеет таймаут и реагирует на отмену. Вспомогательные горутины завершаются вместе с воркером; backoff ограничен 5 минутами.
- Логи: кольцевой буфер на 100 строк без копирования среза на каждую запись. На диске текущий файл и один предыдущий, до 256 КиБ каждый. Ротация синхронизирована, ошибки записи видны в stdout. Для 500 аккаунтов номинальный предел файловых логов — 250 МиБ, не считая stdout Docker/systemd. Уже существующий файл сверх лимита может остаться в предыдущем поколении до следующей ротации. Нет постоянно открытого файла на каждый аккаунт.
- /game не сообщает ложный успех, когда TARGET_GAME из окружения перекрыл бы запись config.env. Бот объясняет, где менять игру.
- HLS-соединения переиспользуются. Лимиты простаивающих соединений: 128 всего, 32 на хост, закрытие через 90 секунд простоя. Общий лимит активных запросов и дополнительная очередь не вводились — согласованный пользователем вариант. При reload простаивающие соединения старого менеджера закрываются.
- Бот принимает любой .txt/.TXT, проверяет содержимое и сохраняет как account.txt. Старый accounts.txt переносится, если нового файла ещё нет; существующий account.txt имеет приоритет. Нестандартные пути из конфига сохраняются.

## 500 аккаунтов

Для одновременной работы 500 аккаунтов они должны находиться в одной группе: группы работают по очереди. Запуск распределён по минуте, шаг для 500 аккаунтов — 120 мс. Лимита 128 аккаунтов нет.

Переиспользование снижает необходимость повторных TCP/TLS-подключений, но не гарантирует скорость ответов Twitch, отсутствие ограничений API или зачисление времени. Доступность следующих наград определяется ответами Twitch. Для подтверждения реального фарма нужен длительный прогон на целевом VPS с проверкой прогресса.

## Проверки

- Полный набор тестов с повторениями и go vet; результаты финального запуска указаны в сообщении о завершении.
- Загрузка .txt: кириллица, пробелы, несколько точек, .TXT, неверные расширения/пути, лимит 2 МиБ, ошибка проверки без reload, сохранность старого файла при некорректной загрузке.
- Перенос старого файла и приоритет существующего account.txt.
- Auth-воркер не перезапускается между сессиями; нет непрерывного опроса при отсутствии дропов; reload прерывает ожидание.
- Telegram: команда не повторяется после пересоздания; оставшиеся сообщения пакета не теряются при отмене.
- Готовый дроп не выбирается для дальнейшего просмотра и учитывается единожды; код автоматического получения удалён.
- 500 клиентов одновременно, по три heartbeat, с задержкой каждого ответа и одним обрывом запроса сегмента на клиента: восстанавливаются HLS и beacon. Это имитация транспорта, не 500 реальных сетевых соединений.
- 1000 клиентов по 20 heartbeat на имитации транспорта.
- Отмена группы из 10 000 воркеров.
- Локальный HTTPS-сервер: 20 запросов плейлиста/HEAD используют одно TCP/TLS-соединение. Отключение проверки сертификата применяется только в тесте.
- Конкурентные записи логов и чтение статуса, порядок кольцевого буфера, ограничение файлов логов, явная ошибка при конфликте TARGET_GAME.
- Fuzz WebSocket за 10 секунд: 510 916 входов без panic, до последней небольшой правки обработки ошибки pong; финальный общий запуск также проверяет seed-корпус.

Race detector не запускался: нет C-компилятора для cgo. Конкурентные тесты не заменяют race detector. Docker daemon недоступен; контейнер и systemd не запускались. Реальные Telegram/Twitch и длительная нагрузка VPS не проверялись.

## Повторение проверок

```powershell
.\review\run.ps1
.\review\run.ps1 -StabilityOnly
$env:GOCACHE = Join-Path $PWD '.gocache'
$env:GOMODCACHE = Join-Path $PWD '.gocache/mod'
go test ./... -count 100 -timeout 120s
go vet ./...
go test ./twitch -run '^$' -fuzz FuzzWebSocketFrames -fuzztime 10s -parallel 4
```

Регрессионные тесты включены в обычный go test ./...; overlay больше не нужен. Они не читают рабочие токены и список аккаунтов и не обращаются к Twitch/Telegram.


## Telegram / Discord control (2026-11-02)

- First interactive launch selects Telegram or Discord and saves the choice; existing complete Telegram installations are adopted. Windows and native VPS use the binary's `--setup` wizard. Docker has equivalent installer prompts and uses `.env`. Service startup does not prompt.
- Exactly one management transport runs. Account workers enqueue notifications without waiting for network I/O. The queue holds 128 entries; bursts are batched and overflow is summarized. Failed deliveries are logged; reward details remain in account logs/status.
- Discord uses one outgoing Gateway WebSocket, zero privileged intents, bounded command queues, deferred ephemeral replies, owner-only direct-message authorization, duplicate-event protection, session resume, missed-ACK detection, and cancellable reconnect/rate-limit waits. Attachment downloads allow only Discord CDN hosts, reject redirects, omit authorization, and enforce the 2 MiB cap before account validation/replacement.
- Added tests for 500 concurrent notification producers (10,000 notifications during a stalled bot request), queue bounds, cancellation, authorization, duplicate commands, HTTP 429/header waits, Gateway resume/invalid session/missing ACK, upload host/size restrictions, first-launch persistence, legacy adoption, and environment priority.
- Full Go suite, repeated runs, `go vet`, module verification, Windows and Linux amd64/arm64 builds pass. PowerShell/Bash syntax checks pass. govulncheck reports no reachable vulnerabilities.
- Live Discord login and command responses were subsequently observed. Slash-upload URL validation was corrected after a real upload failure; prolonged live farming with Discord still needs verification. No native Linux/Docker runtime was available here. Race detector requires an installed C compiler and was not run.


### Discord direct messages

Discord management now uses the owner's bot DM. Global slash commands register only the BOT_DM context; guild, group-chat, and non-owner interactions are rejected. Notification delivery resolves and verifies the owner's DM once per manager lifetime and caches that ID behind a cancellable gate. A failed or mismatched DM result is not cached and never receives notifications. The channel ID setting is no longer required and legacy values are ignored. Setup prompts, templates, Docker installer, and both README languages were updated.

Added regression checks for DM-only registration/authorization, concurrent resolution (20 sends, one create-DM request), invalid recipient/channel responses, cancellation during resolution and gate waits, and first-launch setup without a channel ID. Full tests/vet and repeated Discord/setup tests pass. Windows and Linux amd64/arm64 binaries were rebuilt. Live login and command replies have been observed. The slash-upload path fix passed regression tests; end-to-end upload success and prolonged notification delivery still require a live check.


### v0.1.1 publication checks (2026-11-02)

Slash-command uploads now accept Discord's `/ephemeral-attachments/` CDN path as well as `/attachments/`, including signed query strings and Unicode filenames. Host allowlisting, redirect rejection, omitted CDN authorization, size limits, and account-file validation remain enforced. Added regression coverage through the `/accounts` command.

The latest short live run had a correlated Twitch connection failure and subsequent HLS recovery across affected accounts. This release does not claim the network failure is fixed or that prolonged farming stability has been proven. Mock load tests and build checks do not establish live Twitch crediting rates.


## Independent watch operations (2026-11-03, after v0.1.1)

A live log review showed 101 seconds between successful HLS checks while sequential request timeouts and channel replacement accumulated. HLS/recovery remains owned by the account worker. A sequential beacon loop and a sequential Inventory loop now run independently, with at most one in-flight operation of each type and a single latest-result slot. Parent cancellation cancels requests and joins both loops before returning/reload. A stream-generation context cancels an old beacon during channel replacement; beacon cadence survives replacement. Inventory continues during playback retry pauses. Completed/disappeared/claimed drop results terminate polling and remain available to the owner.

Timeout budgets: 15 seconds for a complete HLS check with refresh/retry, 20 seconds for complete channel selection/recovery, 8 seconds per channel candidate, and 15 seconds each for beacon and Inventory. Failed channels still cool down for five minutes; slow-rate recovery still preserves a working stream if replacement fails. Rate samples from polls begun before playback reset do not measure the new window. HLS logs include elapsed time. Date and time in account logs now come from one captured instant, fixing three observed midnight timestamp inconsistencies.

Regression tests exercise 250/500 accounts with separately stalled HLS, beacon and Inventory, assert continued independent operations and one request of each type per account, and verify all requests are gone after cancellation. Further checks cover switching away from a hanging HLS channel, drop completion while stream opening stalls, cancellation of an old stream's beacon without a replacement burst, midnight timestamps, and the existing cadence/recovery suite. The scenarios use a mocked transport; they do not establish live Twitch crediting or eliminate server/network outages.


Validation for this change: full Go suite (three repetitions), additional isolation/cancellation/cadence scenarios (ten repetitions, followed by five repetitions after expanding completion tests), and go vet passed. A compiler for cgo is not installed, so the race detector was unavailable; concurrent mocked tests are not a substitute for it. Live Twitch performance after this change remains to be measured. No release is being published with this commit.
