FROM node:12

RUN mkdir -p /app

WORKDIR /app

COPY package*.json /app/

RUN npm install

COPY . /app/

USER node

EXPOSE 4200

CMD ["npm", "start"]