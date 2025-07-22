import os
from sqlalchemy import create_engine, Column, Integer, String, ForeignKey
from sqlalchemy.orm import declarative_base, relationship
from dotenv import load_dotenv

load_dotenv()
DATABASE_URL = os.getenv("DATABASE_URL")

engine = create_engine(DATABASE_URL)
Base = declarative_base()


class LongURL(Base):
    __tablename__ = 'long_urls'
    id = Column(Integer, primary_key=True)
    url = Column(String, nullable=False)
    links = relationship("URLLink", back_populates="long_url")


class ShortURL(Base):
    __tablename__ = 'short_urls'
    id = Column(Integer, primary_key=True)
    short_code = Column(String, unique=True, nullable=False)
    click_count = Column(Integer, default=0)
    links = relationship("URLLink", back_populates="short_url")


class URLLink(Base):
    __tablename__ = 'url_links'
    id = Column(Integer, primary_key=True)
    long_url_id = Column(Integer, ForeignKey('long_urls.id'))
    short_url_id = Column(Integer, ForeignKey('short_urls.id'))
    long_url = relationship("LongURL", back_populates="links")
    short_url = relationship("ShortURL", back_populates="links")

if __name__ == "__main__":
    Base.metadata.create_all(engine)
