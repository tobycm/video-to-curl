// funky time

import videoToAscii from "ascii-video";
import { createWriteStream, existsSync, mkdirSync } from "fs";
import http from "http";
import { Writable } from "stream";

const allowedMimeTypes = ["video/mp4"];

if (!existsSync("./temp")) mkdirSync("./temp");

const server = http.createServer();

server.on("request", async (request, response) => {
  response.setHeader("Access-Control-Allow-Origin", "*");
  response.setHeader("Access-Control-Allow-Headers", "*");

  if (request.method == "OPTIONS") return response.writeHead(200).end();

  if (request.url === "/") return response.writeHead(200).end("Hello, World!");

  if (request.url?.startsWith("/makeAscii") && request.method === "POST") {
    const id = request.url.split("/")[2];
    if (!id) return response.writeHead(400).end("No id provided");

    const videoFile = createWriteStream(`./temp/${id}.mp4`);

    try {
      await new Promise<void>((resolve, reject) => {
        request.on("data", (chunk) => {
          videoFile.write(chunk);
          // 1e9 === 1GB
          if (videoFile.bytesWritten > 1e9) reject("Request entity too large");
        });

        request.on("end", resolve);
      });
    } catch {
      return response.writeHead(413).end("Request entity too large");
    }

    // check mime later

    const output = new Writable({
      async write(chunk, encoding, callback) {
        response.write("\x1b[2J");
        response.write(chunk);
        await new Promise((resolve) => setTimeout(resolve, 1000 / 15));
        callback();
      },

      final(callback) {
        response.end("\n\nThx for watching!");
        return callback();
      },
    });

    videoToAscii.create(`./temp/${id}.mp4`, output, { width: 40 });

    return;
  }

  response.writeHead(404).end("Not found");
});

const port = parseInt(process.env.PORT || "3000", 10);

server.listen(port, () => {
  console.log(`Listening on port ${port}`);
});
