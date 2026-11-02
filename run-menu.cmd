@echo off
REM Windows entry: prefer Git Bash / WSL for the real Linux binaries & Sage.
set ROOT=%~dp0
where bash >nul 2>&1
if %ERRORLEVEL%==0 (
  bash "%ROOT%run-menu.sh"
  goto :eof
)
where wsl >nul 2>&1
if %ERRORLEVEL%==0 (
  wsl -e bash "%ROOT%run-menu.sh"
  goto :eof
)
echo This artifact's prebuilt binaries target Linux AMD64.
echo Install Git Bash or WSL, then run:  bash run-menu.sh
pause
