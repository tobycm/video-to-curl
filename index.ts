// funky time

import videoToAscii from "ascii-video";
import { createWriteStream, existsSync, mkdirSync } from "fs";
import http from "http";
import WritableQueue from "./WritableQueue";

const allowedMimeTypes = ["video/mp4"];

if (!existsSync("./temp")) mkdirSync("./temp");

const server = http.createServer();

server.on("request", async (request, response) => {
  response.setHeader("Access-Control-Allow-Origin", "*");
  response.setHeader("Access-Control-Allow-Headers", "*");

  if (request.method == "OPTIONS") return response.writeHead(200).end();

  const url = new URL(request.url || "", `http://${request.headers.host}`);

  if (url.pathname === "/") return response.writeHead(200).end("Hello, World!");

  if (url.pathname.startsWith("/makeAscii") && request.method === "POST") {
    const id = url.pathname.split("/")[2];
    if (!id) return response.writeHead(400).end("No id provided");

    response.write("\nUploading...");

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

    response.write("\nUploaded!");

    // check mime later

    response.write("\nProcessing...");

    const output = new WritableQueue({});

    videoToAscii.create(`./temp/${id}.mp4`, output, {
      width: parseInt(url.searchParams.get("width") ?? "50", 10),
      fps: 10,
    });

    response.write("\nProcessed!\nStarting video");

    for (let i = 0; i < 5; i++) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      response.write(".");
    }

    response.write("\nEnjoy your video!\n\n");

    const loop = setInterval(async () => {
      if (output.queue.length > 0) {
        response.write("\x1b[2J\n");
        response.write(output.queue.shift());
      } else if (output.writableEnded) {
        response.end("\nThx for watching!\n");
        clearInterval(loop);
      } else response.write("\nBuffering...");
    }, 1000 / 10);

    return;
  }

  response.writeHead(404).end("Not found");
});

const port = parseInt(process.env.PORT || "3000", 10);

server.listen(port, () => {
  console.log(`Listening on port ${port}`);
});
