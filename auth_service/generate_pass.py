import bcrypt
import hashlib
import password_module
s = bcrypt.gensalt()

parola_admin = password_module.generate_password("parola_admin")
print(parola_admin)

