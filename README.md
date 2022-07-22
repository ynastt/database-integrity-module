# database-integrity-module
ArangoDB database integrity module

## Модуль проверки целостности базы данных
*Программа реализует проверку целостности базы данных в ArangoDB: верно ли были перенесены данные с Bircoin Core в базу в ArangoDB*

## Конфигурация проекта
- `src/arango` - пакет подключения к ArangoDB
- `src/bitcoin_rpc` - пакет подключения к Bitcoin Core (JSON RPC API)
- `src/check_fields` - пакет проверки полей (кроме ключа) документов коллекций в ArangoDB

## Сборка модуля
- необходимо выполнить команду `bash ./build.sh`
- в каталоге `bin` появится исполняемый файл `main`

## Важно
- Для задания параметров подключения к Bitcoin Core измените константы SERVER_HOST, SERVER_PORT, USER, PASSWD, USESSL в `bitcoin_rpc.go`	
- в пакет go-bitcoind добавлен дополнительный метод `GetRawTransactionUPD` в файл `bitcoind.go`, файл с этой функцией расположен в репозитории. При выполнении сборки программы, небходимо заменить исходный файл `src/github.com/Toorop/go-bitcoind/bitcoind.go` на данный, после замены заново выполните команду сборки `bash ./build.sh`
- при выполнении программы будут созданы текстовые файлы:
    - `txt/transactions.txt` для транзакций с разными типами представления выхода
    - `txt/keys_of_imported_docs.txt`для ключей документов, которые не попали в базу данных при переносе с bitcoin-core
    - `txt/incorrect_fields.txt` для названий коллекций и ключей документов с некорректными полями

## Запуск программы
- в каталоге `bin` выполнить команду `./main`

