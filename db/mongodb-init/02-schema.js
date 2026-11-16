// Schema colectiilor din SP (Tabelele 2-5): validatori $jsonSchema si indexuri.
//
// Ruleaza automat o singura data, la prima pornire a containerului, dupa 01-create-users.js
// (scripturile din /docker-entrypoint-initdb.d ruleaza in ordine alfabetica).
// E scris ca sa poata fi rulat si manual, de cate ori e nevoie: daca o colectie exista deja,
// ii actualizeaza validatorul (collMod), iar createIndex nu face nimic pentru un index existent.

const dbName = process.env.MONGO_DB_NAME;
if (!dbName) {
  throw new Error("Lipseste variabila de mediu MONGO_DB_NAME");
}
const projectDb = db.getSiblingDB(dbName);

// Fragmente refolosite in scheme.
const objectIdList = { bsonType: "array", items: { bsonType: "objectId" } };
const nonEmptyString = { bsonType: "string", minLength: 1 };
const absolutePath = { bsonType: "string", pattern: "^/" }; // SP: "format cale absoluta"

// Fiecare colectie: schema documentelor si indexurile.
// Numele unui index e chiar numele campului: cand MongoDB raporteaza o cheie duplicata
// (eroarea 11000), mesajul contine numele indexului, deci serviciul stie ce camp sa raporteze (409).
const collections = {
  // SP, Tabelul 2
  users: {
    schema: {
      bsonType: "object",
      required: ["id_user_grpc", "userName"],
      additionalProperties: false,
      properties: {
        _id: { bsonType: "objectId" },
        id_user_grpc: {
          bsonType: "string",
          pattern: "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
        },
        userName: nonEmptyString,
        id_comp_clust: {
          bsonType: "array",
          items: {
            bsonType: "object",
            required: ["id", "tip"],
            additionalProperties: false,
            properties: {
              id: { bsonType: "objectId" },
              tip: { enum: ["calculator", "cluster"] },
            },
          },
        },
      },
    },
    indexes: [{ key: { id_user_grpc: 1 }, options: { name: "id_user_grpc", unique: true } }],
  },

  // SP, Tabelul 3
  computers: {
    schema: {
      bsonType: "object",
      required: ["IP"],
      additionalProperties: false,
      properties: {
        _id: { bsonType: "objectId" },
        IP: nonEmptyString,
        hostname: nonEmptyString, // poate lipsi (SP), dar nu poate fi gol
        id_script: objectIdList,
        id_users: objectIdList,
      },
    },
    indexes: [
      { key: { IP: 1 }, options: { name: "IP", unique: true } },
      // Index unic partial: se aplica doar documentelor care au hostname. Fara filtru,
      // MongoDB ar trata toate hostname-urile lipsa ca pe aceeasi valoare (null) si ar refuza
      // al doilea calculator fara hostname.
      {
        key: { hostname: 1 },
        options: { name: "hostname", unique: true, partialFilterExpression: { hostname: { $exists: true } } },
      },
      // Pentru stergerea in cascada a unui utilizator: gaseste calculatoarele care il au in id_users.
      { key: { id_users: 1 }, options: { name: "id_users" } },
    ],
  },

  // SP, Tabelul 4
  clusters: {
    schema: {
      bsonType: "object",
      required: ["nume"],
      additionalProperties: false,
      properties: {
        _id: { bsonType: "objectId" },
        nume: nonEmptyString,
        id_comp: objectIdList,
        id_users: objectIdList,
      },
    },
    indexes: [
      { key: { nume: 1 }, options: { name: "nume", unique: true } },
      // Pentru intrebarea "in ce cluster e acest calculator?" (un calculator e in cel mult unul).
      { key: { id_comp: 1 }, options: { name: "id_comp" } },
      { key: { id_users: 1 }, options: { name: "id_users" } },
    ],
  },

  // SP, Tabelul 6, plus campurile cerute de executorul din student-kit (SP permite: "puteti
  // propune scheme noi"): target (pe ce ruleaza), message si results (scrise de executor).
  // date, state, status si notification pot lipsi (SP); state lipsa = pending.
  tasks: {
    schema: {
      bsonType: "object",
      required: ["id_user", "id_script", "target"],
      additionalProperties: false,
      properties: {
        _id: { bsonType: "objectId" },
        id_user: { bsonType: "objectId" },
        id_script: { bsonType: "objectId" },
        target: {
          bsonType: "object",
          required: ["tip", "id"],
          additionalProperties: false,
          properties: {
            tip: { enum: ["calculator", "cluster"] },
            id: { bsonType: "objectId" },
          },
        },
        date: { bsonType: "date" },
        state: { enum: ["pending", "active", "finished"] },
        status: { bsonType: "string" },
        message: { bsonType: "string" },
        results: { bsonType: "array", items: { bsonType: "object" } }, // un rezultat per calculator
        notification: { bsonType: "bool" },
      },
    },
    indexes: [
      // Istoricul unui maintainer: task-urile lui, in ordinea id-urilor.
      { key: { id_user: 1, _id: 1 }, options: { name: "id_user" } },
      // Executorul cauta periodic task-urile pending ratate (student-kit, RESCAN_SECONDS).
      { key: { state: 1 }, options: { name: "state" } },
    ],
  },

  // SP, Tabelul 5
  scripts: {
    schema: {
      bsonType: "object",
      required: ["name", "store_path", "store_comp"],
      additionalProperties: false,
      properties: {
        _id: { bsonType: "objectId" },
        name: nonEmptyString,
        store_path: absolutePath,
        store_comp: absolutePath,
        is_active: { bsonType: "bool" }, // poate lipsi: echivalent cu false (SP)
      },
    },
    indexes: [],
  },
};

const existing = projectDb.getCollectionNames();
for (const [name, spec] of Object.entries(collections)) {
  // strict + error: orice insert sau update care nu respecta schema e refuzat.
  const validation = {
    validator: { $jsonSchema: spec.schema },
    validationLevel: "strict",
    validationAction: "error",
  };
  if (existing.includes(name)) {
    projectDb.runCommand({ collMod: name, ...validation });
  } else {
    projectDb.createCollection(name, validation);
  }
  for (const index of spec.indexes) {
    projectDb.getCollection(name).createIndex(index.key, index.options);
  }
  print(`Colectia ${name}: schema si ${spec.indexes.length} indexuri.`);
}
