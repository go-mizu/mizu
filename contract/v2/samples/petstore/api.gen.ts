import { Petstore } from "@acme/petstore"

const client = new Petstore({
  baseUrl: "https://api.example.com",
  apiKey: process.env.PETSTORE_API_KEY!,
})

// 1) List pets (GET /v1/pets?limit=...&cursor=...)
const page1 = await client.pets.list({ limit: 20 })
for (const p of page1.data) {
  console.log(p.id, p.name, p.status)
}

// 2) Create pet (POST /v1/pets)
const pet = await client.pets.create({
  name: "Mochi",
  status: "available",
  tags: [{ name: "cute" }],
})

// 3) Retrieve pet (GET /v1/pets/{id})
const same = await client.pets.retrieve(pet.id)

// 4) Update pet (PATCH /v1/pets/{id})
const updated = await client.pets.update({
  id: pet.id,
  status: "sold",
})

// 5) Delete pet (DELETE /v1/pets/{id})
await client.pets.delete({ id: pet.id })

// 6) Inventory (GET /v1/store/inventory)
const inv = await client.stores.getInventory()
console.log(inv["available"], inv["sold"])

// 7) Place order (POST /v1/store/orders)
const order = await client.stores.placeOrder({
  pet_id: pet.id,
  quantity: 2,
})

// 8) Retrieve order (GET /v1/store/orders/{id})
const got = await client.stores.retrieveOrder({ id: order.id })

// 9) Delete order (DELETE /v1/store/orders/{id})
await client.stores.deleteOrder({ id: order.id })

// 10) Users (POST /v1/users, GET /v1/users/{username})
const user = await client.users.create({ username: "alice", email: "alice@example.com" })
const alice = await client.users.retrieve({ username: "alice" })
console.log(alice.username)
