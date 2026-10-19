import bcrypt

s = bcrypt.gensalt()

parola = bcrypt.hashpw("parola".encode(), salt= s)
print(parola.decode())
