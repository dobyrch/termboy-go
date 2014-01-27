Term Boy
========

Term Boy is a Nintendo Game Boy emulator...for your terminal.

This is an early implementation written in Go. A more complete version (written
in C++) is available [here](https://github.com/dobyrch/termboy).

This project is based on code from an existing Game Boy Color emulator,
[gomeboycolor](https://github.com/djhworld/gomeboycolor).

![Boot](screenshots/screen_0.png)
![Intro](screenshots/screen_1.png)
![Title](screenshots/screen_2.png)
![Gameplay](screenshots/screen_3.png)

Usage
-----

After running `go install`, start Term Boy by running `termboy-go <ROM.gb>` in a
Linux virtual console. Use ESDF for the D-pad, G/H for SELECT/START, and J/K for
B/A.  Press ESC to quit.

Miscellanea
-----------

Ubuntu users may see the message "Failed to set font height."  Term Boy uses
the `setfont` command to change the font height, which looks for the font
*default8x16.psfu* in /usr/share/consolefonts.  The font can be downloaded from
the [Kbd project](http://kbd-project.org/download/).  Download any of the
archives and the font will be located in data/consolefonts.

An branch for FreeBSD is also available (`git checkout freebsd`).  See PORTING
for more details.

Sound is not yet supported.  If you want sound now, try out my other
[implementation](https://github.com/dobyrch/termboy).
