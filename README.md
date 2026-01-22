# Финальный проект 1 семестра

REST API сервис для загрузки и выгрузки данных о ценах.

## Требования к системе

- Go 1.23 
- PostgreSQL 15
- Bash shell 
- curl 

## Установка и запуск

```bash
git clone https://github.com/pavliks/itmo-devops-sem1-project-template.git
cd itmo-devops-sem1-project-template
```
- Подготовка приложения
```bash
./scripts/prepare.sh
```
- Запуск приложения
```bash
./scripts/run.sh
```

## Тестирование
- Тестирование
```bash
./scripts/tests.sh 1
```

## API Эндпоинты

### POST /api/v0/prices

Загрузка данных о ценах из ZIP архива в базу данных.

**Параметры запроса:**
- `file` - ZIP архив с файлом `data.csv` (multipart/form-data)

**Пример запроса:**
```bash
curl -F "file=@sample_data.zip" http://localhost:8080/api/v0/prices
```

**Формат ответа:**
```json
{
  "total_items": 12,
  "total_categories": 5,
  "total_price": 4740.39
}
```

### GET /api/v0/prices

Выгрузка всех данных из базы данных в виде ZIP архива.

**Пример запроса:**
```bash
curl http://localhost:8080/api/v0/prices -o data.zip
```

**Формат ответа:**
- Content-Type: `application/zip`
- Content-Disposition: `attachment; filename=data.zip`


Директория `sample_data` - это пример директории, которая является разархивированной версией файла `sample_data.zip`

Какие тесты проходит приложение? Можно предоставить команду для тестов или описание тестов со скришотами/видео.

## Контакт

К кому можно обращаться в случае вопросов?
