// funky time

import videoToAscii from "ascii-video";
import { createWriteStream, existsSync, mkdirSync } from "fs";
import http from "http";

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

    const frames = await videoToAscii.create(`./temp/${id}.mp4`, { width: 160 });

    console.log(frames.length);

    for (const frame of frames) {
      response.write("\x1b[2J");
      response.write(frame);
      await new Promise((resolve) => setTimeout(resolve, 1000 / 15));
    }

    return response.end("\n\nThx for watching!");
  }

  response.writeHead(404).end("Not found");
});

const port = parseInt(process.env.PORT || "3000", 10);

server.listen(port, () => {
  console.log(`Listening on port ${port}`);
});
