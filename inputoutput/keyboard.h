#ifndef KEYBOARD
#define KEYBOARD

#include <fcntl.h>
#include <unistd.h>
#include <termios.h>
#include <sys/kbio.h>
#include <sys/ioctl.h>

static struct termios tty_old;
static int kbd_old;
static int initialized = 0;

//TODO: make this cross platform
//TODO: write in Go instead of C
int kbd_init() {
	struct termios tty_attr;
	int flags;

	/* make stdin non-blocking */
	flags = fcntl(STDIN_FILENO, F_GETFL);
	flags |= O_NONBLOCK;
	fcntl(STDIN_FILENO, F_SETFL, flags);

	/* save old keyboard mode */
	if (ioctl(STDIN_FILENO, KDGKBMODE, &kbd_old) < 0) {
		return 0;
	}

	tcgetattr(STDIN_FILENO, &tty_old);

	/* turn off buffering, echo and key processing */
	tty_attr = tty_old;
	tty_attr.c_lflag &= ~(ICANON | ECHO | ISIG);
	tty_attr.c_iflag &= ~(ISTRIP | INLCR | ICRNL | IGNCR | IXON | IXOFF);
	tcsetattr(STDIN_FILENO, TCSANOW, &tty_attr);

	ioctl(STDIN_FILENO, KDSKBMODE, K_RAW);

	initialized = 1;
	return 1;
}

void kbd_restore() {
	if (initialized) {
		tcsetattr(STDIN_FILENO, TCSAFLUSH, &tty_old);
		ioctl(STDIN_FILENO, KDSKBMODE, kbd_old);
	}
}

#endif
