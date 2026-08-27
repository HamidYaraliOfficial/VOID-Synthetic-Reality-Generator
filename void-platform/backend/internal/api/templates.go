package api

// Template describes one built-in, customizable Scenario starting point
// from the Template Library (E-Commerce, Banking, IoT, ...).
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Description string `json:"description"`
}

func builtinTemplates() []Template {
	return []Template{
		{ID: "ecommerce", Name: "E-Commerce Marketplace", Domain: "retail", Description: "Customers, Orders, Products, Payments, Inventory with checkout behaviors."},
		{ID: "banking", Name: "Digital Banking", Domain: "fintech", Description: "Accounts, Transactions, Cards, fraud-style business rules."},
		{ID: "iot", Name: "IoT Fleet", Domain: "iot", Description: "Devices, Sensors, Telemetry events, connectivity chaos scenarios."},
		{ID: "saas", Name: "SaaS Product", Domain: "software", Description: "Users, Sessions, Subscriptions, feature-usage behaviors."},
		{ID: "social", Name: "Social Network", Domain: "social", Description: "Users, Posts, Follows, engagement + notification events."},
		{ID: "logistics", Name: "Logistics & Shipping", Domain: "logistics", Description: "Shipments, Vehicles, Warehouses, route/delay simulation."},
		{ID: "smartcity", Name: "Smart City", Domain: "iot", Description: "Sensors, Vehicles, Network Nodes across a city-scale topology."},
		{ID: "fintech", Name: "FinTech Payments", Domain: "fintech", Description: "Payments, Ledgers, Risk rules, settlement transactions."},
		{ID: "microservices", Name: "Microservices Mesh", Domain: "infrastructure", Description: "Services, Databases, Caches, dependency + chaos graph."},
		{ID: "gaming", Name: "Gaming Backend", Domain: "gaming", Description: "Players, Matches, Sessions, in-game economy transactions."},
		{ID: "telemetry", Name: "Telemetry Pipeline", Domain: "observability", Description: "Devices, Events, high-throughput ingestion load profiles."},
	}
}
