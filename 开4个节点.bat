@echo off
for /L %%i in (1,1,4) do start client.exe -p 1000%%i -s 192.168.1.8:30000
