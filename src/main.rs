use actix_web::{post, App, HttpResponse, HttpServer, Responder, web, HttpRequest};
use mongodb::{bson::doc, options::ClientOptions, Client, Collection};
use std::env;

#[post("/upload")]
async fn upload_video(collection: web::Data<Collection>) -> impl Responder {
    HttpResponse::Ok().body("Not implemented yet.")
}


#[post("/delete")]
async fn delete_video(req: HttpRequest, collection: web::Data<Collection>) -> impl Responder {
    let auth_user_id = get_content_type(&req, "auth_user_id").unwrap().to_owned().parse::<u64>().unwrap();
    let video_id = get_content_type(&req, "video_id").unwrap().to_owned().parse::<u64>().unwrap();
    
    let video = collection.find_one(doc!{"_id": video_id}, None);

    HttpResponse::Ok().body("Ok!")
}

 fn get_content_type<'a>(req: &'a HttpRequest, header: &str) -> Option<&'a str> {
    req.headers().get(header)?.to_str().ok()
}


#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let mut client_options =
        ClientOptions::parse(&env::var("mongo_uri").unwrap().to_string())
            .await.unwrap();
    client_options.app_name = Some("Videos service".to_string());


    HttpServer::new( move || {
        App::new()
            .data(Client::with_options(client_options.clone()).unwrap().database("blue").collection("video_metadata")) // Wtf is this shit
            .service(upload_video)
            .service(delete_video)
    })
    .bind("0.0.0.0:".to_string() + &env::var("PORT").unwrap_or_else(|_| {panic!("PORT env var not supplied")}))?
    .run()
    .await
}
