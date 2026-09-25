# Calculator frontend

React + TypeScript + Vite UI for the calculator API.

## Run locally

```sh
npm install
npm run dev
```

The app expects the backend API at `/api/calculate`. Run the Go backend on the
same origin or configure the local development server with a proxy as needed.

## Verify

```sh
npm test
npm run build
```

The calculator supports addition, subtraction, multiplication, division,
exponentiation, square root, and percentage. Arithmetic is performed by the
backend; the frontend only validates input, sends operands, and formats the
returned result.
