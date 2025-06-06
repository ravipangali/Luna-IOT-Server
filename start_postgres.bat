@echo off
echo Starting PostgreSQL service...

REM Start PostgreSQL service (if installed as Windows service)
net start postgresql-x64-14

REM Alternative for different PostgreSQL versions:
REM net start postgresql-x64-13
REM net start postgresql-x64-15
REM net start postgresql-x64-16

REM If PostgreSQL is not installed as a service, you might need to run:
REM "C:\Program Files\PostgreSQL\14\bin\pg_ctl.exe" -D "C:\Program Files\PostgreSQL\14\data" start

echo PostgreSQL service started!
echo.
echo Now you can run:
echo   go run cmd/http-server/main.go
echo   go run cmd/tcp-server/main.go
echo.
pause 