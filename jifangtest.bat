@echo off
for /L %%i in (1,1,8) do start client.exe -p 1800%%i -s 172.16.6.5:30000
