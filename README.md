# database-integrity-module
ArangoDB database integrity module

## Модуль проверки целостности базы данных
*Программа реализует проверку целостности базы данных в ArangoDB: верно ли были перенесены данные с bircoin-core в базу в в ArangoDB*

## Важно для установки
- скачать используемые пакеты
	`"github.com/arangodb/go-driver"`
	`"github.com/arangodb/go-driver/http"`
	`"github.com/Toorop/go-bitcoind"`
- в пакет go-bitcoind добавлена дополнительная функция `GetRawTransactionUPD` в файл `bitcoind.go`, файл с этой функции расположен в репозитории
- при выполнении программы будут созданы два текстовых файла:
	`transactions.txt` для транзакций с разными типами представления выхода
	`keys_of_imported_docs.txt`для ключей документов, которые не попали в базу данных при переносе с bitcoin-core
	
